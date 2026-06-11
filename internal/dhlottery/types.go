// Package dhlottery implements the DHLottery domain: authentication, account
// info, deposit/charge, lotto purchase, pension reserve, and winning checks.
package dhlottery

// AccountInfo holds the account's deposit balance and current lotto round.
type AccountInfo struct {
	Balance      int
	CurrentRound int
}

// PurchaseReadyResponse is the payload from /egovUserReadySocket.json.
type PurchaseReadyResponse struct {
	DirectYN  string `json:"direct_yn"`
	ReadyIP   string `json:"ready_ip"`
	ReadyTime string `json:"ready_time"`
	ReadyCnt  string `json:"ready_cnt"`
}

// gameSelection is one game in the execBuy param array. ArrGameChoiceNum is a
// nil slice so it marshals to JSON null for auto-generated games.
type gameSelection struct {
	GenType          string `json:"genType"`
	ArrGameChoiceNum []int  `json:"arrGameChoiceNum"`
	Alpabet          string `json:"alpabet"`
}

// purchaseResult is the execBuy.do response.
type purchaseResult struct {
	LoginYN string `json:"loginYn"`
	Result  struct {
		ResultCode string `json:"resultCode"`
		ResultMsg  string `json:"resultMsg"`
	} `json:"result"`
}

// PurchaseOutcome is the result of a purchase attempt (success or failure).
type PurchaseOutcome struct {
	Success      bool
	RoundNumber  int
	GameCount    int
	TotalAmount  int
	PurchaseDate string
	Message      string
	Error        string
	Code         string
}

// WinningResult is a winning row from the ledger API. Rank is 0 when the API
// omits wnRnk. Product is the lottery name (e.g. 로또6/45, 연금복권720+).
type WinningResult struct {
	RoundNumber int
	Rank        int
	PrizeAmount int
	Product     string
	WinResult   string
}

// LedgerSummary is the lifetime aggregate recomputed from the full ledger:
// cumulative purchase (Σ prchsQty × CostPerGame) and cumulative winning
// (Σ ltWnAmt where ltWnAmt > 0).
type LedgerSummary struct {
	CumulativePurchase int
	CumulativeWinning  int
}

// EL (pension 720+) API response types.

type elEncryptedResponse struct {
	Q string `json:"q"`
}

type elResultBase struct {
	ResultCode string `json:"resultCode"`
	ResultMsg  string `json:"resultMsg"`
}

type elRoundRemainTimeResponse struct {
	elResultBase
	Round    string `json:"ROUND"`
	DrawDate string `json:"DRAW_DATE"`
}

type elDepositResponse struct {
	elResultBase
	Deposit string `json:"deposit"`
}

type elDuplicateRound struct {
	DoubleRound string `json:"doubleRound"`
	DoubleCnt   string `json:"doubleCnt"`
}

type elCheckMyReserveResponse struct {
	elResultBase
	DoubleRound []elDuplicateRound `json:"doubleRound"`
}

type elAddMyReserveResponse struct {
	elResultBase
	ReserveOrderNo   string `json:"reserveOrderNo"`
	ReserveOrderDate string `json:"reserveOrderDate"`
}

// PensionReserveOutcome is the result of a pension reserve attempt.
type PensionReserveOutcome struct {
	Status           string // "success" | "skipped" | "failure"
	Success          bool
	Skipped          bool
	TargetRound      int
	HasTargetRound   bool
	TotalAmount      int
	TicketCount      int
	Message          string
	ReserveOrderNo   string
	ReserveOrderDate string
	DuplicateRounds  []string
	Error            string
	Code             string
}
