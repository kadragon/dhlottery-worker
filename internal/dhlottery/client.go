package dhlottery

import (
	"time"

	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

// Client is the public facade over the DHLottery domain operations. It owns an
// HTTP client (with session cookies) and a notification collector.
type Client struct {
	http      *httpclient.Client
	collector *notify.Collector
}

// NewClient returns a Client with a fresh HTTP client and collector.
func NewClient() *Client {
	return &Client{
		http:      httpclient.New(),
		collector: &notify.Collector{},
	}
}

// Collector returns the notification collector.
func (c *Client) Collector() *notify.Collector { return c.collector }

// Login authenticates with DHLottery.
func (c *Client) Login() error { return login(c.http) }

// GetAccountInfo returns the balance and current lottery round.
func (c *Client) GetAccountInfo() (AccountInfo, error) { return getAccountInfo(c.http) }

// CheckDeposit checks the balance and initializes a charge if insufficient.
func (c *Client) CheckDeposit(requiredAmount int) (bool, error) {
	return checkDeposit(c.http, requiredAmount, c.collector)
}

// ReservePensionNextWeek reserves next week's pension 720+ ticket.
func (c *Client) ReservePensionNextWeek() PensionReserveOutcome {
	return reservePensionNextWeek(c.http, c.collector)
}

// Buy purchases 5 auto-pick lottery games.
func (c *Client) Buy() PurchaseOutcome { return purchaseLottery(c.http, c.collector) }

// CheckWinning checks the previous week's winning results.
func (c *Client) CheckWinning(now time.Time) []WinningResult {
	return checkWinning(c.http, now)
}

// ProbeLedgerRange is a TEMPORARY diagnostic: reports how many ledger rows the
// server returns for [strDt, endDt] (YYYYMMDD). TODO: remove after chunking.
func (c *Client) ProbeLedgerRange(strDt, endDt string) LedgerProbe {
	return probeLedgerRange(c.http, strDt, endDt)
}

// DumpLedgerRange is a TEMPORARY diagnostic returning page-1 raw rows of
// [strDt, endDt] (YYYYMMDD). TODO: remove after LP72 cost is confirmed.
func (c *Client) DumpLedgerRange(strDt, endDt string) ([]LedgerRowSample, bool) {
	return dumpLedgerRange(c.http, strDt, endDt)
}

// AggregateLedger recomputes lifetime cumulative purchase and winning totals
// from the full ledger over [startDate, now]. ok is false when the ledger
// lookup failed (network/parse/redirect/non-200).
func (c *Client) AggregateLedger(startDate string, now time.Time) (LedgerSummary, bool) {
	return aggregateLedger(c.http, startDate, now)
}
