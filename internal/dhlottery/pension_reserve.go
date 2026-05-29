package dhlottery

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/datekst"
	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/format"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

const (
	elBaseURL          = "https://el.dhlottery.co.kr"
	totalGameURL       = elBaseURL + "/game/TotalGame.jsp?LottoId=LP72"
	reservePageURL     = elBaseURL + "/game/pension720/reserveGame.jsp"
	roundRemainTimeURL = elBaseURL + "/roundRemainTime.do"
	checkDepositURL    = elBaseURL + "/checkDeposit.do"
	checkMyReserveURL  = elBaseURL + "/checkMyReserve.do"
	addMyReserveURL    = elBaseURL + "/addMyReserve.do"
	// Empty frmAuto serialization accepted as-is for session-scoped queries.
	frmAutoSerialized = "ROUND=&SEL_NO=&BUY_CNT=&AUTO_SEL_SET=&SEL_CLASS=&ACCS_TYPE=01"
	ticketCount       = 5
)

type roundInfo struct {
	currentRound    int
	nextRound       int
	nextDrawDateDot string
}

func createAjaxHeaders(referer string) map[string]string {
	return map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"User-Agent":       constants.UserAgent,
		"Origin":           elBaseURL,
		"Referer":          referer,
		"X-Requested-With": "XMLHttpRequest",
	}
}

func getSessionID(client *httpclient.Client) (string, error) {
	sessionID := client.Cookie("JSESSIONID")
	key := "JSESSIONID"
	if sessionID == "" {
		sessionID = client.Cookie("DHJSESSIONID")
		key = "DHJSESSIONID"
	}
	if sessionID == "" {
		return "", dherr.New("Missing session cookie for pension reserve", "PENSION_AUTH_MISSING")
	}
	logger.Debug("Using session cookie for EL encryption", logger.Fields{"cookieKey": key})
	return sessionID, nil
}

func toDotDate(ymd string) string {
	return strings.ReplaceAll(ymd, "-", ".")
}

func buildReserveFormPayload(ri roundInfo, deposit int, winDate string) string {
	v := url.Values{}
	v.Set("ROUND", strconv.Itoa(ri.currentRound))
	v.Set("reserveJo", "0")
	v.Set("repeatRoundCnt", "1")
	v.Set("totalBuyAmt", strconv.Itoa(constants.PensionReserveCost))
	v.Set("totalBuyCnt", strconv.Itoa(ticketCount))
	v.Set("moneyBalance", "0/") // trailing slash is the site's serialized format
	v.Set("couponBalance", "0/")
	v.Set("nextRound", strconv.Itoa(ri.nextRound))
	v.Set("repeatClass", "5")
	v.Set("roundBuyCnt", "1")
	v.Set("curdeposit", strconv.Itoa(deposit))
	v.Set("curpay", strconv.Itoa(constants.PensionReserveCost))
	v.Set("winDate", winDate)
	v.Set("WORKING_FLAG", "false")
	v.Set("repeatRound", "1")
	v.Set("repeatRoundHidden", ri.nextDrawDateDot)
	return v.Encode()
}

func bootstrapElSession(client *httpclient.Client) error {
	totalResp, err := client.Fetch(totalGameURL, httpclient.RequestOptions{
		Headers: map[string]string{"User-Agent": constants.UserAgent},
	})
	if err != nil {
		return err
	}
	if totalResp.Status != 200 {
		return dherr.New(fmt.Sprintf("Failed to load EL total game page: HTTP %d", totalResp.Status), "PENSION_BOOTSTRAP_FAILED")
	}

	reserveResp, err := client.Fetch(reservePageURL, httpclient.RequestOptions{
		Headers: map[string]string{"User-Agent": constants.UserAgent, "Referer": totalGameURL},
	})
	if err != nil {
		return err
	}
	if reserveResp.Status != 200 {
		return dherr.New(fmt.Sprintf("Failed to load EL reserve page: HTTP %d", reserveResp.Status), "PENSION_BOOTSTRAP_FAILED")
	}
	return nil
}

func fetchRoundRemainTime(client *httpclient.Client) (roundInfo, error) {
	resp, err := client.Fetch(roundRemainTimeURL, httpclient.RequestOptions{
		Method:  "POST",
		Headers: createAjaxHeaders(reservePageURL),
		Body:    frmAutoSerialized,
	})
	if err != nil {
		return roundInfo{}, err
	}
	if resp.Status != 200 {
		return roundInfo{}, dherr.New(fmt.Sprintf("Failed to fetch round remain time: HTTP %d", resp.Status), "PENSION_ROUND_FETCH_FAILED")
	}

	var data elRoundRemainTimeResponse
	if err := resp.JSON(&data); err != nil {
		return roundInfo{}, err
	}
	if data.ResultCode != "100" {
		return roundInfo{}, dherr.New(fmt.Sprintf("Round remain time API failed: %s %s", data.ResultCode, data.ResultMsg), data.ResultCode)
	}

	currentRound, err := strconv.Atoi(data.Round)
	if err != nil || currentRound <= 0 {
		return roundInfo{}, dherr.New("Invalid round value: "+data.Round, "PENSION_INVALID_ROUND")
	}

	nextDrawDate := datekst.AddDaysToYmd(data.DrawDate, 7)
	return roundInfo{
		currentRound:    currentRound,
		nextRound:       currentRound + 1, // pension draws are weekly and sequential
		nextDrawDateDot: toDotDate(nextDrawDate),
	}, nil
}

func postEncrypted(client *httpclient.Client, endpoint, plainForm, referer string, out any) error {
	sessionID, err := getSessionID(client)
	if err != nil {
		return err
	}
	encryptedQ, err := EncryptElQ(plainForm, sessionID)
	if err != nil {
		return err
	}

	resp, err := client.Fetch(endpoint, httpclient.RequestOptions{
		Method:  "POST",
		Headers: createAjaxHeaders(referer),
		Body:    url.Values{"q": {encryptedQ}}.Encode(),
	})
	if err != nil {
		return err
	}
	if resp.Status != 200 {
		return dherr.New(fmt.Sprintf("EL API request failed: HTTP %d", resp.Status), "PENSION_API_FAILED")
	}

	var enc elEncryptedResponse
	if err := resp.JSON(&enc); err != nil {
		return err
	}
	if enc.Q == "" {
		return dherr.New("EL API response is missing q payload", "PENSION_API_INVALID_RESPONSE")
	}

	decrypted, err := DecryptElQ(enc.Q, sessionID)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(decrypted), out)
}

func createFailure(message, code string, targetRound int, hasTargetRound bool) PensionReserveOutcome {
	return PensionReserveOutcome{
		Status:         "failure",
		Success:        false,
		Skipped:        false,
		TargetRound:    targetRound,
		HasTargetRound: hasTargetRound,
		Error:          message,
		Code:           code,
	}
}

// verifyDeposit returns the deposit, or a failure/skip outcome, or an error
// (transport/crypto failures bubble to the top-level handler).
func verifyDeposit(client *httpclient.Client, targetRound int, collector *notify.Collector) (int, *PensionReserveOutcome, error) {
	var depositData elDepositResponse
	if err := postEncrypted(client, checkDepositURL, frmAutoSerialized, reservePageURL, &depositData); err != nil {
		return 0, nil, err
	}

	if depositData.ResultCode != "100" {
		notify.Notify(notify.Payload{
			Type:    notify.Error,
			Title:   "연금복권 예약 실패",
			Message: "연금복권 예치금 조회에 실패했습니다: " + depositData.ResultMsg,
			Details: []notify.KV{
				{Key: "오류코드", Value: depositData.ResultCode},
				{Key: "대상회차", Value: fmt.Sprintf("%d회", targetRound)},
			},
		}, collector)
		f := createFailure(depositData.ResultMsg, depositData.ResultCode, targetRound, true)
		return 0, &f, nil
	}

	deposit, err := strconv.Atoi(depositData.Deposit)
	if err != nil {
		notify.Notify(notify.Payload{
			Type:    notify.Error,
			Title:   "연금복권 예약 실패",
			Message: "연금복권 예치금 값을 파싱하지 못했습니다.",
		}, collector)
		f := createFailure("Invalid deposit value from EL API", "PENSION_INVALID_DEPOSIT", 0, false)
		return 0, &f, nil
	}

	if deposit < constants.PensionReserveCost {
		notify.Notify(notify.Payload{
			Type:    notify.Warning,
			Title:   "연금복권 예약 건너뜀",
			Message: "연금복권 예약에 필요한 예치금이 부족하여 예약을 건너뜁니다.",
			Details: []notify.KV{
				{Key: "대상회차", Value: fmt.Sprintf("%d회", targetRound)},
				{Key: "필요금액", Value: format.KoreanNumber(constants.PensionReserveCost) + "원"},
				{Key: "보유예치금", Value: format.KoreanNumber(deposit) + "원"},
			},
		}, collector)
		f := createFailure("Insufficient balance for pension reserve", "PENSION_INSUFFICIENT_DEPOSIT", targetRound, true)
		return 0, &f, nil
	}

	return deposit, nil, nil
}

func checkForDuplicates(client *httpclient.Client, ri roundInfo, deposit int, collector *notify.Collector) (*PensionReserveOutcome, error) {
	payload := buildReserveFormPayload(ri, deposit, "")
	var dupData elCheckMyReserveResponse
	if err := postEncrypted(client, checkMyReserveURL, payload, reservePageURL, &dupData); err != nil {
		return nil, err
	}

	if dupData.ResultCode != "100" {
		notify.Notify(notify.Payload{
			Type:    notify.Error,
			Title:   "연금복권 예약 실패",
			Message: "연금복권 중복 예약 확인에 실패했습니다: " + dupData.ResultMsg,
			Details: []notify.KV{
				{Key: "오류코드", Value: dupData.ResultCode},
				{Key: "대상회차", Value: fmt.Sprintf("%d회", ri.nextRound)},
			},
		}, collector)
		f := createFailure(dupData.ResultMsg, dupData.ResultCode, ri.nextRound, true)
		return &f, nil
	}

	var duplicates []string
	for _, item := range dupData.DoubleRound {
		if n, _ := strconv.Atoi(item.DoubleRound); n == ri.nextRound {
			duplicates = append(duplicates, fmt.Sprintf("%s회 %s매", item.DoubleRound, item.DoubleCnt))
		}
	}

	if len(duplicates) > 0 {
		notify.Notify(notify.Payload{
			Type:    notify.Warning,
			Title:   "연금복권 예약 건너뜀",
			Message: fmt.Sprintf("대상 회차(%d회)가 이미 예약되어 예약을 건너뜁니다.", ri.nextRound),
			Details: []notify.KV{{Key: "중복회차", Value: strings.Join(duplicates, ", ")}},
		}, collector)
		skipped := PensionReserveOutcome{
			Status:          "skipped",
			Success:         true,
			Skipped:         true,
			TargetRound:     ri.nextRound,
			HasTargetRound:  true,
			TotalAmount:     constants.PensionReserveCost,
			TicketCount:     ticketCount,
			Message:         "Duplicate reserve detected",
			DuplicateRounds: duplicates,
		}
		return &skipped, nil
	}

	return nil, nil
}

func submitReservation(client *httpclient.Client, ri roundInfo, deposit int, collector *notify.Collector) (PensionReserveOutcome, error) {
	payload := buildReserveFormPayload(ri, deposit, ri.nextDrawDateDot)
	var data elAddMyReserveResponse
	if err := postEncrypted(client, addMyReserveURL, payload, reservePageURL, &data); err != nil {
		return PensionReserveOutcome{}, err
	}

	targetRound := ri.nextRound

	if data.ResultCode != "100" {
		notify.Notify(notify.Payload{
			Type:    notify.Error,
			Title:   "연금복권 예약 실패",
			Message: "연금복권 예약 요청에 실패했습니다: " + data.ResultMsg,
			Details: []notify.KV{
				{Key: "오류코드", Value: data.ResultCode},
				{Key: "대상회차", Value: fmt.Sprintf("%d회", targetRound)},
			},
		}, collector)
		return createFailure(data.ResultMsg, data.ResultCode, targetRound, true), nil
	}

	notify.Notify(notify.Payload{
		Type:    notify.Success,
		Title:   "연금복권 예약 완료",
		Message: fmt.Sprintf("%d회 연금복권 예약을 완료했습니다.", targetRound),
		Details: []notify.KV{
			{Key: "대상회차", Value: fmt.Sprintf("%d회", targetRound)},
			{Key: "예약금액", Value: format.KoreanNumber(constants.PensionReserveCost) + "원"},
			{Key: "예약번호", Value: data.ReserveOrderNo},
		},
	}, collector)

	return PensionReserveOutcome{
		Status:           "success",
		Success:          true,
		Skipped:          false,
		TargetRound:      targetRound,
		HasTargetRound:   true,
		TotalAmount:      constants.PensionReserveCost,
		TicketCount:      ticketCount,
		Message:          data.ResultMsg,
		ReserveOrderNo:   data.ReserveOrderNo,
		ReserveOrderDate: data.ReserveOrderDate,
	}, nil
}

// reservePensionNextWeek reserves next week's pension 720+ ticket. It never
// returns an error; failures are reported via the outcome and a notification.
func reservePensionNextWeek(client *httpclient.Client, collector *notify.Collector) PensionReserveOutcome {
	var targetRound int
	var hasTarget bool

	outcome, err := func() (PensionReserveOutcome, error) {
		if err := bootstrapElSession(client); err != nil {
			return PensionReserveOutcome{}, err
		}

		ri, err := fetchRoundRemainTime(client)
		if err != nil {
			return PensionReserveOutcome{}, err
		}
		targetRound = ri.nextRound
		hasTarget = true

		deposit, fail, err := verifyDeposit(client, targetRound, collector)
		if err != nil {
			return PensionReserveOutcome{}, err
		}
		if fail != nil {
			return *fail, nil
		}

		dup, err := checkForDuplicates(client, ri, deposit, collector)
		if err != nil {
			return PensionReserveOutcome{}, err
		}
		if dup != nil {
			return *dup, nil
		}

		return submitReservation(client, ri, deposit, collector)
	}()

	if err != nil {
		code := dherr.Code(err)
		if code == "" {
			code = "PENSION_UNEXPECTED_ERROR"
		}
		details := []notify.KV{{Key: "오류코드", Value: code}}
		if hasTarget {
			details = append(details, notify.KV{Key: "대상회차", Value: fmt.Sprintf("%d회", targetRound)})
		}
		notify.Notify(notify.Payload{
			Type:    notify.Error,
			Title:   "연금복권 예약 실패",
			Message: "연금복권 예약 중 오류가 발생했습니다: " + err.Error(),
			Details: details,
		}, collector)
		return createFailure(err.Error(), code, targetRound, hasTarget)
	}

	return outcome
}
