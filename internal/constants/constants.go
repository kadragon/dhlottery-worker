// Package constants holds business-rule constants for the DHLottery worker.
package constants

const (
	// MinDepositAmount is the minimum deposit required for a lottery purchase
	// (5 games x 1,000 KRW = 5,000 KRW, no safety buffer).
	MinDepositAmount = 5000

	// ChargeAmount is the manual deposit (charge) amount in KRW.
	ChargeAmount = 50000

	// GamesPerPurchase is the number of lottery games bought per run.
	GamesPerPurchase = 5

	// CostPerGame is the cost of a single game in KRW.
	CostPerGame = 1000

	// TotalPurchaseCost is the total lottery purchase cost per run in KRW.
	TotalPurchaseCost = GamesPerPurchase * CostPerGame

	// PensionReserveCost is the pension 720+ reserve amount per run in KRW.
	PensionReserveCost = 5000

	// WeeklyCombinedRequiredBalance is the minimum balance to run both the
	// pension reserve and the lotto purchase.
	WeeklyCombinedRequiredBalance = TotalPurchaseCost + PensionReserveCost

	// PurchaseSuccessCode is the resultCode returned by execBuy.do on success.
	PurchaseSuccessCode = "100"

	// DefaultLedgerStartDate (YYYYMMDD) is the lifetime ledger query start used
	// when the LEDGER_START_DATE env var is unset.
	DefaultLedgerStartDate = "20200101"

	// UserAgent is the browser-like User-Agent sent with every request.
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36"

	// HTTP request header names and common values used in DHLottery API calls.
	HeaderUserAgent           = "User-Agent"
	HeaderReferer             = "Referer"
	HeaderContentType         = "Content-Type"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderXRequestedWithValue = "XMLHttpRequest"
)
