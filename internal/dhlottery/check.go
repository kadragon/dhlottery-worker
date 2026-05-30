package dhlottery

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/datekst"
	"github.com/kadragon/dhlottery-worker/internal/format"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/notify"
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

// checkWinning fetches the previous week's ledger and notifies on every win
// (lotto and pension). Non-fatal by design: network/parse errors and 3xx
// redirects return an empty slice.
func checkWinning(client *httpclient.Client, now time.Time, collector *notify.Collector) []WinningResult {
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

	wins := extractWins(data.Data.List)
	for _, win := range wins {
		details := []notify.KV{
			{Key: "product", Value: win.Product},
			{Key: "roundNumber", Value: strconv.Itoa(win.RoundNumber)},
		}
		if win.Rank > 0 {
			details = append(details, notify.KV{Key: "rank", Value: strconv.Itoa(win.Rank)})
		}
		if win.WinResult != "" {
			details = append(details, notify.KV{Key: "winResult", Value: win.WinResult})
		}
		details = append(details,
			notify.KV{Key: "prizeAmount", Value: strconv.Itoa(win.PrizeAmount)},
			notify.KV{Key: "prizeAmountKrw", Value: format.KoreanNumber(win.PrizeAmount) + "원"},
			notify.KV{Key: "period", Value: r.StartDate + " ~ " + r.EndDate},
		)

		notify.Notify(notify.Payload{
			Type:    notify.Success,
			Title:   "복권 당첨!",
			Message: fmt.Sprintf("%s %d회차 당첨을 확인했습니다.", win.Product, win.RoundNumber),
			Details: details,
		}, collector)
	}

	return wins
}
