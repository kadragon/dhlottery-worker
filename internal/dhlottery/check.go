package dhlottery

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/datekst"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
)

// 2026-01: the legacy lottoBuyList HTML page (myPage.do?method=lottoBuyList)
// was retired and now 302-redirects to /errorPage. Purchase/winning history is
// served by this JSON ledger API, which covers both lotto (LO40) and pension
// (LP72) in one feed.
const winningLedgerURL = "https://www.dhlottery.co.kr/mypage/selectMyLotteryledger.do"

// ledgerResponse is the /mypage/selectMyLotteryledger.do payload.
type ledgerResponse struct {
	Data struct {
		Total int         `json:"total"`
		List  []ledgerRow `json:"list"`
	} `json:"data"`
}

// ledgerRow is one purchase/winning record. LtWnAmt distinguishes the three
// states: nil = not yet drawn, 0 = lost, > 0 = won.
type ledgerRow struct {
	LtGdsCd    string `json:"ltGdsCd"`
	LtGdsNm    string `json:"ltGdsNm"`
	LtEpsd     int    `json:"ltEpsd"`
	LtWnResult string `json:"ltWnResult"`
	LtWnAmt    *int   `json:"ltWnAmt"`
	WnRnk      *int   `json:"wnRnk"`
	EpsdRflDt  string `json:"epsdRflDt"`
	PrchsQty   int    `json:"prchsQty"`
}

// extractWins keeps rows with a positive win amount. A nil LtWnAmt (undrawn) or
// zero (lost) is skipped, so a win is detected on LtWnAmt > 0 rather than on
// wnRnk — whose encoding for winning rows is not observable from lost/undrawn
// data, and which is therefore unsafe to gate the notification on.
func extractWins(rows []ledgerRow) []WinningResult {
	var wins []WinningResult
	for _, row := range rows {
		if row.LtWnAmt == nil || *row.LtWnAmt <= 0 {
			continue
		}
		rank := 0
		if row.WnRnk != nil {
			rank = *row.WnRnk
		}
		wins = append(wins, WinningResult{
			RoundNumber: row.LtEpsd,
			Rank:        rank,
			PrizeAmount: *row.LtWnAmt,
			Product:     row.LtGdsNm,
			WinResult:   row.LtWnResult,
		})
	}
	return wins
}

func compactYmd(date string) string { return strings.ReplaceAll(date, "-", "") }

// checkWinning fetches the previous week's ledger and returns its wins (lotto
// and pension). Non-fatal by design: network/parse errors and 3xx redirects
// return an empty slice. Notification is handled by the caller via the weekly
// settlement summary, so this function does not push payloads.
func checkWinning(client *httpclient.Client, now time.Time) []WinningResult {
	r := datekst.CalculatePreviousWeekRange(now)

	u, err := url.Parse(winningLedgerURL)
	if err != nil {
		logger.Error("Winning check failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "winning_check_failed", logger.FieldError: err.Error(),
		})
		return nil
	}
	q := u.Query()
	q.Set("srchStrDt", compactYmd(r.StartDate))
	q.Set("srchEndDt", compactYmd(r.EndDate))
	q.Set("sort", "")
	q.Set("ltGdsCd", "")
	q.Set("winResult", "")
	q.Set("lramSmam", "")
	q.Set("pageNum", "1")
	q.Set("recordCountPerPage", "50")
	u.RawQuery = q.Encode()

	resp, err := client.Fetch(u.String(), httpclient.RequestOptions{
		Headers: map[string]string{
			constants.HeaderUserAgent:      constants.UserAgent,
			"Accept":                       "application/json, text/javascript, */*; q=0.01",
			constants.HeaderContentType:    "application/json;charset=UTF-8",
			constants.HeaderXRequestedWith: constants.HeaderXRequestedWithValue,
			"ajax":                         "true",
			constants.HeaderReferer:        "https://www.dhlottery.co.kr/mypage/mylotteryledger",
		},
	})
	if err != nil {
		logger.Error("Winning check failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "winning_check_failed", logger.FieldError: err.Error(),
		})
		return nil
	}

	// redirect: 'manual' means a 3xx is not success — usually an expired
	// session. Only 200 returns parseable JSON.
	if resp.Status != 200 {
		isRedirect := resp.Status >= 300 && resp.Status < 400
		fields := logger.Fields{logger.FieldStatus: resp.Status}
		if isRedirect {
			fields[logger.FieldEvent] = "winning_fetch_redirect"
			fields["location"] = resp.Header.Get("Location")
		} else {
			fields[logger.FieldEvent] = "winning_fetch_failed"
		}
		logger.Error("Winning fetch failed", fields)
		return nil
	}

	var data ledgerResponse
	if err := resp.JSON(&data); err != nil {
		logger.Error("Winning check failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "winning_check_failed", logger.FieldError: err.Error(),
		})
		return nil
	}

	return extractWins(data.Data.List)
}

// aggregateLedger recomputes lifetime totals from the full ledger over
// [startDate, now]. Cumulative purchase = Σ(prchsQty × CostPerGame);
// cumulative winning = Σ(ltWnAmt where > 0). Pages through all rows using
// data.total. Non-fatal by design: any fetch/parse/redirect/non-200 error
// returns a zero LedgerSummary.
func aggregateLedger(client *httpclient.Client, startDate string, now time.Time) LedgerSummary {
	const perPage = 100
	const maxPages = 200 // backstop for a server that ignores pageNum / returns a bogus total

	endDate := compactYmd(datekst.FormatKstYmd(now))

	var purchase, winning, fetched, total int
	for page := 1; page <= maxPages; page++ {
		data, ok := fetchLedgerPage(client, compactYmd(startDate), endDate, page, perPage)
		if !ok {
			return LedgerSummary{}
		}
		if page == 1 {
			total = data.Data.Total
		}
		for _, row := range data.Data.List {
			purchase += row.PrchsQty * constants.CostPerGame
			if row.LtWnAmt != nil && *row.LtWnAmt > 0 {
				winning += *row.LtWnAmt
			}
		}
		fetched += len(data.Data.List)
		if len(data.Data.List) == 0 || fetched >= total {
			break
		}
	}

	return LedgerSummary{CumulativePurchase: purchase, CumulativeWinning: winning}
}

// fetchLedgerPage fetches one page of the ledger. Returns ok=false (after
// logging) on any network/parse error, redirect, or non-200 status.
func fetchLedgerPage(client *httpclient.Client, strDt, endDt string, page, perPage int) (ledgerResponse, bool) {
	var data ledgerResponse

	u, err := url.Parse(winningLedgerURL)
	if err != nil {
		logger.Error("Ledger aggregate failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "ledger_aggregate_failed", logger.FieldError: err.Error(),
		})
		return data, false
	}
	q := u.Query()
	q.Set("srchStrDt", strDt)
	q.Set("srchEndDt", endDt)
	q.Set("sort", "")
	q.Set("ltGdsCd", "")
	q.Set("winResult", "")
	q.Set("lramSmam", "")
	q.Set("pageNum", strconv.Itoa(page))
	q.Set("recordCountPerPage", strconv.Itoa(perPage))
	u.RawQuery = q.Encode()

	resp, err := client.Fetch(u.String(), httpclient.RequestOptions{
		Headers: map[string]string{
			constants.HeaderUserAgent:      constants.UserAgent,
			"Accept":                       "application/json, text/javascript, */*; q=0.01",
			constants.HeaderContentType:    "application/json;charset=UTF-8",
			constants.HeaderXRequestedWith: constants.HeaderXRequestedWithValue,
			"ajax":                         "true",
			constants.HeaderReferer:        "https://www.dhlottery.co.kr/mypage/mylotteryledger",
		},
	})
	if err != nil {
		logger.Error("Ledger aggregate failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "ledger_aggregate_failed", logger.FieldError: err.Error(),
		})
		return data, false
	}
	if resp.Status != 200 {
		logger.Error("Ledger aggregate fetch failed", logger.Fields{
			logger.FieldEvent: "ledger_aggregate_fetch_failed", logger.FieldStatus: resp.Status,
		})
		return data, false
	}
	if err := resp.JSON(&data); err != nil {
		logger.Error("Ledger aggregate failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "ledger_aggregate_failed", logger.FieldError: err.Error(),
		})
		return data, false
	}
	return data, true
}
