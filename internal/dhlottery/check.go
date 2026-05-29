package dhlottery

import (
	"fmt"
	"net/url"
	"regexp"
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

const winningListURL = "https://www.dhlottery.co.kr/myPage.do?method=lottoBuyList"

// Winning-results HTML parsing patterns.
var (
	rowRe        = regexp.MustCompile(`(?is)<tr\b.*?</tr>`)
	tableCellRe  = regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td>`)
	thRe         = regexp.MustCompile(`(?i)<th\b`)
	detailPopRe  = regexp.MustCompile(`(?i)detailPop\(\s*'[^']*'\s*,\s*'[^']*'\s*,\s*'(\d+)'\s*\)`)
	digitRe      = regexp.MustCompile(`(\d+)`)
	matchCountRe = regexp.MustCompile(`(\d+)\s*개`)
	htmlTagRe    = regexp.MustCompile(`<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
	nonDigitRe   = regexp.MustCompile(`[^\d]`)
)

func stripHTMLTags(input string) string {
	s := htmlTagRe.ReplaceAllString(input, " ")
	s = whitespaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func extractTdTexts(rowHTML string) []string {
	matches := tableCellRe.FindAllStringSubmatch(rowHTML, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, stripHTMLTags(m[1]))
	}
	return out
}

// parseWinningResultsFromHTML parses winning rows from lottoBuyList HTML.
// Row layout: 0 buyDate, 1 gameName, 2 detail, 3 count, 4 result, 5 prize, 6 drawDate.
func parseWinningResultsFromHTML(html string) []WinningResult {
	var results []WinningResult
	for _, row := range rowRe.FindAllString(html, -1) {
		if thRe.MatchString(row) {
			continue
		}
		cells := extractTdTexts(row)
		if len(cells) < 6 {
			continue
		}

		issue := detailPopRe.FindStringSubmatch(row)
		roundOK := issue != nil
		roundNumber := 0
		if roundOK {
			roundNumber, _ = strconv.Atoi(issue[1])
		}

		resultCell := cells[4]
		rankMatch := digitRe.FindStringSubmatch(resultCell)
		rankOK := rankMatch != nil
		rank := 0
		if rankOK {
			rank, _ = strconv.Atoi(rankMatch[1])
		}

		var matchCount *int
		if mc := matchCountRe.FindStringSubmatch(resultCell); mc != nil {
			v, _ := strconv.Atoi(mc[1])
			matchCount = &v
		}

		prizeDigits := nonDigitRe.ReplaceAllString(cells[5], "")
		prizeOK := prizeDigits != ""
		prize := 0
		if prizeOK {
			prize, _ = strconv.Atoi(prizeDigits)
		}

		if !roundOK || !rankOK || !prizeOK {
			continue
		}
		if rank < 1 || prize < 0 {
			continue
		}

		results = append(results, WinningResult{
			RoundNumber: roundNumber,
			Rank:        rank,
			PrizeAmount: prize,
			MatchCount:  matchCount,
		})
	}
	return results
}

// filterJackpotWins keeps rank 1 (jackpot) wins only.
func filterJackpotWins(results []WinningResult) []WinningResult {
	var out []WinningResult
	for _, r := range results {
		if r.Rank == 1 {
			out = append(out, r)
		}
	}
	return out
}

// checkWinning fetches the previous week's results and notifies jackpot wins.
// Non-fatal by design: network/parse errors return an empty slice.
func checkWinning(client *httpclient.Client, now time.Time, collector *notify.Collector) []WinningResult {
	r := datekst.CalculatePreviousWeekRange(now)

	u, err := url.Parse(winningListURL)
	if err != nil {
		logger.Error("Winning check failed (non-fatal)", logger.Fields{
			"event": "winning_check_failed", "error": err.Error(),
		})
		return nil
	}
	q := u.Query()
	q.Set("searchStartDate", r.StartDate)
	q.Set("searchEndDate", r.EndDate)
	q.Set("nowPage", "1")
	u.RawQuery = q.Encode()

	resp, err := client.Fetch(u.String(), httpclient.RequestOptions{
		Headers: map[string]string{"User-Agent": constants.UserAgent},
	})
	if err != nil {
		logger.Error("Winning check failed (non-fatal)", logger.Fields{
			"event": "winning_check_failed", "error": err.Error(),
		})
		return nil
	}

	// redirect: 'manual' means a 3xx is not success — it usually signals an
	// expired session. Only 200 returns parseable HTML.
	if resp.Status != 200 {
		isRedirect := resp.Status >= 300 && resp.Status < 400
		fields := logger.Fields{"status": resp.Status}
		if isRedirect {
			fields["event"] = "winning_fetch_redirect"
			fields["location"] = resp.Header.Get("Location")
		} else {
			fields["event"] = "winning_fetch_failed"
		}
		logger.Error("Winning fetch failed", fields)
		return nil
	}

	html, err := resp.Text("euc-kr")
	if err != nil {
		logger.Error("Winning check failed (non-fatal)", logger.Fields{
			"event": "winning_check_failed", "error": err.Error(),
		})
		return nil
	}

	jackpotWins := filterJackpotWins(parseWinningResultsFromHTML(html))
	if len(jackpotWins) == 0 {
		return nil
	}

	for _, win := range jackpotWins {
		details := []notify.KV{
			{Key: "roundNumber", Value: strconv.Itoa(win.RoundNumber)},
			{Key: "rank", Value: strconv.Itoa(win.Rank)},
			{Key: "prizeAmount", Value: strconv.Itoa(win.PrizeAmount)},
			{Key: "prizeAmountKrw", Value: format.KoreanNumber(win.PrizeAmount) + "원"},
		}
		if win.MatchCount != nil {
			details = append(details, notify.KV{Key: "matchCount", Value: strconv.Itoa(*win.MatchCount)})
		}
		details = append(details, notify.KV{Key: "period", Value: r.StartDate + " ~ " + r.EndDate})

		notify.Notify(notify.Payload{
			Type:    notify.Success,
			Title:   "로또 당첨!",
			Message: fmt.Sprintf("%d회차 로또 %d등 당첨을 확인했습니다.", win.RoundNumber, win.Rank),
			Details: details,
		}, collector)
	}

	return jackpotWins
}
