package dhlottery

import (
	"fmt"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
)

// 2026-01: the homepage is JS-rendered, so account info is read from JSON APIs:
//   - /lt645/selectThsLt645Info.do  -> current lotto round (public)
//   - /mypage/selectUserMndp.do     -> available balance (authenticated)
const (
	lottoRoundAPIURL = "https://www.dhlottery.co.kr/lt645/selectThsLt645Info.do"
	balanceAPIURL    = "https://www.dhlottery.co.kr/mypage/selectUserMndp.do"
	accountModule    = "account"
)

// getAccountInfo fetches the current round and available balance.
func getAccountInfo(client *httpclient.Client) (AccountInfo, error) {
	logger.Debug("Fetching lotto round API", logger.Fields{logger.FieldModule: accountModule, "url": lottoRoundAPIURL})

	roundResp, err := client.Fetch(lottoRoundAPIURL, httpclient.RequestOptions{
		Headers: map[string]string{constants.HeaderUserAgent: constants.UserAgent},
	})
	if err != nil {
		return AccountInfo{}, err
	}
	if roundResp.Status != 200 {
		return AccountInfo{}, dherr.New(
			fmt.Sprintf("Account fetch failed at round API (url: %s): HTTP %d%s",
				lottoRoundAPIURL, roundResp.Status, locationInfo(roundResp)),
			"ACCOUNT_FETCH_FAILED")
	}

	currentRound, err := parseRoundFromAPI(roundResp)
	if err != nil {
		return AccountInfo{}, err
	}

	logger.Debug("Fetching balance API", logger.Fields{logger.FieldModule: accountModule, "url": balanceAPIURL})

	balanceResp, err := client.Fetch(balanceAPIURL, httpclient.RequestOptions{
		Headers: map[string]string{
			constants.HeaderUserAgent: constants.UserAgent,
			constants.HeaderReferer:   "https://www.dhlottery.co.kr/mypage/home",
		},
	})
	if err != nil {
		return AccountInfo{}, err
	}
	if balanceResp.Status != 200 {
		return AccountInfo{}, dherr.New(
			fmt.Sprintf("Account fetch failed at balance API (url: %s): HTTP %d%s",
				balanceAPIURL, balanceResp.Status, locationInfo(balanceResp)),
			"ACCOUNT_FETCH_FAILED")
	}

	balance, err := parseBalanceFromAPI(balanceResp)
	if err != nil {
		return AccountInfo{}, err
	}

	if err := validateAccountInfo(balance, currentRound); err != nil {
		return AccountInfo{}, err
	}

	return AccountInfo{Balance: balance, CurrentRound: currentRound}, nil
}

func locationInfo(resp *httpclient.Response) string {
	if loc := resp.Header.Get("Location"); loc != "" {
		return " (Location: " + loc + ")"
	}
	return ""
}

func parseRoundFromAPI(resp *httpclient.Response) (int, error) {
	var data struct {
		Data struct {
			Result struct {
				LtEpsd   int    `json:"ltEpsd"`
				LtRflYmd string `json:"ltRflYmd"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := resp.JSON(&data); err != nil {
		return 0, dherr.New("Failed to parse lottery round from API: "+err.Error(), "ACCOUNT_PARSE_ROUND_FAILED")
	}
	round := data.Data.Result.LtEpsd
	if round <= 0 {
		return 0, dherr.New("Failed to parse lottery round from API: missing or invalid ltEpsd", "ACCOUNT_PARSE_ROUND_FAILED")
	}
	logger.Debug("Found current round from API", logger.Fields{
		logger.FieldModule: accountModule, "round": round, "drawDate": data.Data.Result.LtRflYmd,
	})
	return round, nil
}

func parseBalanceFromAPI(resp *httpclient.Response) (int, error) {
	var data struct {
		Data struct {
			UserMndp struct {
				CrntEntrsAmt *int `json:"crntEntrsAmt"`
			} `json:"userMndp"`
		} `json:"data"`
	}
	if err := resp.JSON(&data); err != nil {
		return 0, dherr.New("Failed to parse balance from API: "+err.Error(), "ACCOUNT_PARSE_BALANCE_FAILED")
	}
	if data.Data.UserMndp.CrntEntrsAmt == nil {
		return 0, dherr.New("Failed to parse balance from API: Missing crntEntrsAmt in API response", "ACCOUNT_PARSE_BALANCE_FAILED")
	}
	balance := *data.Data.UserMndp.CrntEntrsAmt
	logger.Debug("Found balance from API", logger.Fields{logger.FieldModule: accountModule, "balance": balance})
	return balance, nil
}

func validateAccountInfo(balance, currentRound int) error {
	if balance < 0 {
		return dherr.New(fmt.Sprintf("Invalid balance: %d (must be >= 0)", balance), "ACCOUNT_INVALID_DATA")
	}
	if currentRound <= 0 {
		return dherr.New(fmt.Sprintf("Invalid round: %d (must be > 0)", currentRound), "ACCOUNT_INVALID_DATA")
	}
	return nil
}
