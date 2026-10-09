package dhlottery

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/checkpoint"
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

// dashYmd converts YYYYMMDD (or YYYY-MM-DD) to YYYY-MM-DD.
func dashYmd(date string) string { return datekst.AddDaysToYmd(date, 0) }

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

// ledgerWindowDays is the per-query date span. The ledger API silently returns
// an empty list (200, total=0) when srchStrDt..srchEndDt exceeds its limit
// (empirically between 90 and 180 days; 90 confirmed working). sumRange
// therefore walks the range in non-overlapping windows of this size.
const ledgerWindowDays = 90

// ledgerMaxPages is the per-window paging backstop for a server that ignores
// pageNum or returns a bogus data.total.
const ledgerMaxPages = 200

// ledgerRetryDelay is the backoff before the single retry of a failed ledger
// page fetch. ledgerSleep is the seam tests stub.
const ledgerRetryDelay = 500 * time.Millisecond

var ledgerSleep = time.Sleep

// ledgerSettleLagDays is how far behind today the checkpoint's settled cutoff
// trails. The ledger filters by order date, so a row ordered near today can
// still be undrawn (ltWnAmt=null; a pension reserve draws ~10 days after
// ordering). Only rows at least this old are folded into the checkpoint, so a
// later win is never lost to an incremental query that no longer covers it.
const ledgerSettleLagDays = 35

// aggregateLedger recomputes lifetime totals from the full ledger over
// [startDate, now]. Cumulative purchase = Σ(prchsQty × CostPerGame); cumulative
// winning = Σ(ltWnAmt where > 0). Non-fatal by design: on any
// fetch/parse/redirect/non-200 error it returns ok=false (all-or-nothing) so
// the caller can report the lookup failure instead of presenting a partial or
// zero summary as if it were complete.
func aggregateLedger(client *httpclient.Client, startDate string, now time.Time) (LedgerSummary, bool) {
	start, ok := validLedgerStart(startDate)
	if !ok {
		return LedgerSummary{}, false
	}
	end := compactYmd(datekst.FormatKstYmd(now))
	if start > end {
		return LedgerSummary{}, true // start in the future: genuinely nothing to sum
	}
	purchase, winning, ok := sumRange(client, start, end)
	if !ok {
		return LedgerSummary{}, false
	}
	return LedgerSummary{CumulativePurchase: purchase, CumulativeWinning: winning}, true
}

// aggregateLedgerIncremental computes the same lifetime totals as
// aggregateLedger, but resumes from prev when it is a valid checkpoint for
// startDate, querying only (prev.Through, now]. An absent or invalid prev
// falls back to a full scan from startDate. It returns the checkpoint to
// persist — totals over [startDate, today-ledgerSettleLagDays] — or nil when
// nothing has settled yet or on failure (ok=false, all-or-nothing).
func aggregateLedgerIncremental(client *httpclient.Client, startDate string, now time.Time, prev *checkpoint.Checkpoint) (LedgerSummary, *checkpoint.Checkpoint, bool) {
	start, ok := validLedgerStart(startDate)
	if !ok {
		return LedgerSummary{}, nil, false
	}
	today, settled := ledgerCutoffs(now)
	if start > today {
		return LedgerSummary{}, nil, true // start in the future: genuinely nothing to sum
	}

	from := start
	var purchase, winning int
	var base *checkpoint.Checkpoint
	if validCheckpoint(prev, start, settled) {
		base = prev
		// Compact first: datekst's dashed-date parser assumes well-placed dashes.
		from = compactYmd(datekst.AddDaysToYmd(compactYmd(prev.Through), 1))
		purchase, winning = prev.Purchase, prev.Winning
	} else if prev != nil {
		logger.Warn("Ledger checkpoint invalid; falling back to full scan", logger.Fields{
			logger.FieldEvent: "checkpoint_invalid", "start": prev.Start, "through": prev.Through,
		})
	}

	next := base
	if settled >= start && from <= settled {
		p, w, ok := sumRange(client, from, settled)
		if !ok {
			return LedgerSummary{}, nil, false
		}
		purchase += p
		winning += w
		// An empty settled delta (no purchases, or a silently empty response)
		// does not advance the checkpoint: the span is re-queried next run, so
		// a bad read is never frozen in. Changing how rows are summed here or in
		// aggregateWindow invalidates stored totals: bump checkpoint.schemaVersion.
		if p > 0 {
			next = &checkpoint.Checkpoint{
				Start:    dashYmd(start),
				Through:  dashYmd(settled),
				Purchase: purchase,
				Winning:  winning,
			}
		}
	}
	if settled >= from {
		from = compactYmd(datekst.AddDaysToYmd(settled, 1))
	}

	p, w, ok := sumRange(client, from, today)
	if !ok {
		return LedgerSummary{}, nil, false
	}
	return LedgerSummary{CumulativePurchase: purchase + p, CumulativeWinning: winning + w}, next, true
}

// ledgerCutoffs returns today and the settled cutoff (today −
// ledgerSettleLagDays), both YYYYMMDD in KST.
func ledgerCutoffs(now time.Time) (today, settled string) {
	today = compactYmd(datekst.FormatKstYmd(now))
	return today, compactYmd(datekst.AddDaysToYmd(today, -ledgerSettleLagDays))
}

// CheckpointResumable reports whether aggregateLedgerIncremental would resume
// from cp rather than fall back to a full scan, so a caller comparing the two
// paths (cmd/realtest) can tell a real resume from a second full scan.
func CheckpointResumable(cp *checkpoint.Checkpoint, startDate string, now time.Time) bool {
	start, ok := validLedgerStart(startDate)
	if !ok {
		return false
	}
	_, settled := ledgerCutoffs(now)
	return validCheckpoint(cp, start, settled)
}

// validLedgerStart returns startDate as YYYYMMDD. A malformed
// LEDGER_START_DATE (e.g. "foo", unpadded "2026-6-1") must not slip through
// the lexical date comparisons and yield a zero summary tagged as real, so
// anything that is not a valid YYYYMMDD date is rejected.
func validLedgerStart(startDate string) (string, bool) {
	start := compactYmd(startDate)
	if _, err := time.Parse("20060102", start); err != nil {
		logger.Error("Ledger aggregate failed (non-fatal)", logger.Fields{
			logger.FieldEvent: "ledger_invalid_start", logger.FieldError: err.Error(),
		})
		return "", false
	}
	return start, true
}

// validCheckpoint reports whether cp can seed an incremental run: same start
// date, a well-formed Through within [start, settled], and non-negative totals.
// A Through past the settled cutoff may have folded in undrawn rows.
func validCheckpoint(cp *checkpoint.Checkpoint, start, settled string) bool {
	if cp == nil || compactYmd(cp.Start) != start || cp.Purchase < 0 || cp.Winning < 0 {
		return false
	}
	through := compactYmd(cp.Through)
	if _, err := time.Parse("20060102", through); err != nil {
		return false
	}
	return through >= start && through <= settled
}

// sumRange sums purchase and winning over [start, end] (YYYYMMDD, start ≤ end).
// The span is walked in ledgerWindowDays windows, newest first (the API caps a
// single query's date range), each window paged via data.total. Returns
// ok=false if any window fails.
func sumRange(client *httpclient.Client, start, end string) (purchase, winning int, ok bool) {
	const maxWindows = 400 // backstop (~98 years) against a pathological loop

	winEnd := end
	for w := 0; w < maxWindows; w++ {
		winStart := compactYmd(datekst.AddDaysToYmd(winEnd, -(ledgerWindowDays - 1)))
		if winStart < start {
			winStart = start
		}

		p, win, ok := aggregateWindow(client, winStart, winEnd)
		if !ok {
			return 0, 0, false
		}
		purchase += p
		winning += win

		if winStart <= start {
			// Reached the start date: the only legitimate completion.
			return purchase, winning, true
		}
		winEnd = compactYmd(datekst.AddDaysToYmd(winStart, -1)) // next window ends the day before
	}

	// Backstop exhausted without reaching start: the accumulated total is
	// partial, so report failure rather than presenting it as a complete sum.
	logger.Error("Ledger aggregate incomplete (non-fatal)", logger.Fields{
		logger.FieldEvent: "ledger_backstop_exhausted", logger.FieldStatus: maxWindows,
	})
	return 0, 0, false
}

// aggregateWindow sums purchase and winning over a single [strDt, endDt] window,
// paging through all rows via data.total. Returns ok=false on any fetch error
// (after one retry), when the ledgerMaxPages backstop is exhausted before
// data.total rows were read, or when data.total contradicts the rows (rows with
// a missing/zero total, or an empty page before total rows arrived), so a
// truncated sum is never reported as complete.
func aggregateWindow(client *httpclient.Client, strDt, endDt string) (purchase, winning int, ok bool) {
	const perPage = 100

	var fetched, total int
	for page := 1; page <= ledgerMaxPages; page++ {
		data, fetchOK := fetchLedgerPageWithRetry(client, strDt, endDt, page, perPage)
		if !fetchOK {
			return 0, 0, false
		}
		rows := len(data.Data.List)
		if page == 1 {
			total = data.Data.Total
			if total == 0 && rows > 0 {
				logTotalAnomaly(true, "total_zero_with_rows", strDt, endDt, page, rows, fetched, total)
				return 0, 0, false
			}
		} else if data.Data.Total != total {
			logTotalAnomaly(false, "total_changed", strDt, endDt, page, rows, fetched, data.Data.Total)
			total = data.Data.Total // later checks follow the server's latest count
		}
		for _, row := range data.Data.List {
			purchase += row.PrchsQty * constants.CostPerGame
			if row.LtWnAmt != nil && *row.LtWnAmt > 0 {
				winning += *row.LtWnAmt
			}
		}
		if rows == 0 && fetched < total {
			logTotalAnomaly(true, "empty_page_before_total", strDt, endDt, page, rows, fetched, total)
			return 0, 0, false
		}
		fetched += rows
		if total > 0 && fetched > total {
			logTotalAnomaly(false, "rows_exceed_total", strDt, endDt, page, rows, fetched, total)
		}
		if len(data.Data.List) == 0 || fetched >= total {
			return purchase, winning, true
		}
	}

	logger.Error("Ledger aggregate incomplete (non-fatal)", logger.Fields{
		logger.FieldEvent: "ledger_window_truncated",
		"srchStrDt":       strDt,
		"srchEndDt":       endDt,
		"fetched":         fetched,
		"total":           total,
		"maxPages":        ledgerMaxPages,
	})
	return 0, 0, false
}

// logTotalAnomaly records a page whose data.total disagrees with its rows
// (rows on this page, fetched across the window so far, total as reported).
// Realtest confirmed data.total tracks the row count, so an anomaly that means
// rows went missing fails the window (logged as error); one that loses no rows
// (over-delivery, a drifting total) is only a warning.
func logTotalAnomaly(fails bool, reason, strDt, endDt string, page, rows, fetched, total int) {
	log := logger.Warn
	if fails {
		log = logger.Error
	}
	log("Ledger total inconsistent with rows", logger.Fields{
		logger.FieldEvent: "ledger_total_anomaly",
		"reason":          reason,
		"srchStrDt":       strDt,
		"srchEndDt":       endDt,
		"page":            page,
		"rows":            rows,
		"fetched":         fetched,
		"total":           total,
	})
}

// fetchLedgerPageWithRetry retries a transiently failed page fetch (transport
// error or httpclient.TransientStatus) once after ledgerRetryDelay, so a single
// blip does not fail the whole all-or-nothing aggregation. Permanent failures
// (other statuses, redirects, unparseable bodies) fail without a retry.
func fetchLedgerPageWithRetry(client *httpclient.Client, strDt, endDt string, page, perPage int) (ledgerResponse, bool) {
	var data ledgerResponse
	var ok bool
	httpclient.Retry([]time.Duration{ledgerRetryDelay}, ledgerSleep, func(_ int, final bool) (bool, time.Duration) {
		var transient bool
		data, ok, transient = fetchLedgerPage(client, strDt, endDt, page, perPage, !final)
		return !ok && transient, 0
	})
	return data, ok
}

// fetchLedgerPage fetches one page of the ledger. Returns ok=false on any
// network/parse error, redirect, or non-200 status; transient reports whether
// the failure is worth retrying. A transient failure with willRetry set is
// logged as a warning (ledger_retry_attempt); every other failure as an error.
func fetchLedgerPage(client *httpclient.Client, strDt, endDt string, page, perPage int, willRetry bool) (data ledgerResponse, ok, transient bool) {
	fail := func(transient bool, event string, fields logger.Fields) (ledgerResponse, bool, bool) {
		fields["srchStrDt"], fields["srchEndDt"], fields["page"] = strDt, endDt, page
		if transient && willRetry {
			fields[logger.FieldEvent] = "ledger_retry_attempt"
			logger.Warn("Ledger page fetch failed, retrying", fields)
		} else {
			fields[logger.FieldEvent] = event
			logger.Error("Ledger aggregate failed (non-fatal)", fields)
		}
		return data, false, transient
	}

	u, err := url.Parse(winningLedgerURL)
	if err != nil {
		return fail(false, "ledger_aggregate_failed", logger.Fields{logger.FieldError: err.Error()})
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
		return fail(true, "ledger_aggregate_failed", logger.Fields{logger.FieldError: err.Error()})
	}
	if resp.Status != 200 {
		return fail(httpclient.TransientStatus(resp.Status), "ledger_aggregate_fetch_failed",
			logger.Fields{logger.FieldStatus: resp.Status})
	}
	if err := resp.JSON(&data); err != nil {
		return fail(false, "ledger_aggregate_failed", logger.Fields{logger.FieldError: err.Error()})
	}
	return data, true, false
}
