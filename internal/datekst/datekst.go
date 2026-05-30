// Package datekst provides KST (UTC+9) aware date helpers.
//
// The instant is shifted by the KST offset and read with UTC field accessors,
// reproducing the getUTC* arithmetic of the original implementation.
package datekst

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	kstOffset = 9 * time.Hour
	oneDay    = 24 * time.Hour
)

// PreviousWeekRange is a Monday 00:00 KST .. Sunday 23:59:59.999 KST window.
type PreviousWeekRange struct {
	Start     time.Time
	End       time.Time
	StartDate string // YYYY-MM-DD in KST
	EndDate   string // YYYY-MM-DD in KST
}

// FormatKstYmd formats an instant as YYYY-MM-DD in KST.
func FormatKstYmd(t time.Time) string {
	kst := t.Add(kstOffset).UTC()
	y, m, d := kst.Date()
	return formatYmd(y, int(m), d)
}

// CalculatePreviousWeekRange returns the previous week's Monday 00:00 KST to
// Sunday 23:59:59.999 KST window for the given instant.
func CalculatePreviousWeekRange(now time.Time) PreviousWeekRange {
	kstNow := now.Add(kstOffset).UTC()
	y, m, d := kstNow.Date()
	dow := int(kstNow.Weekday()) // 0=Sun .. 6=Sat in the KST-shifted frame
	daysSinceMonday := (dow + 6) % 7

	// Start of the current week (Mon 00:00 KST) as a real instant.
	startOfCurrentWeek := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).
		Add(-kstOffset).
		Add(-time.Duration(daysSinceMonday) * oneDay)

	start := startOfCurrentWeek.Add(-7 * oneDay)
	end := start.Add(7*oneDay - time.Millisecond)

	return PreviousWeekRange{
		Start:     start,
		End:       end,
		StartDate: FormatKstYmd(start),
		EndDate:   FormatKstYmd(end),
	}
}

// NextSaturdayKst returns the next Saturday (today if already Saturday) in KST,
// formatted as YYYY-MM-DD. Lottery draws happen Saturdays 20:00 KST.
func NextSaturdayKst(now time.Time) string {
	kstNow := now.Add(kstOffset).UTC()
	dow := int(kstNow.Weekday())

	var daysUntilSaturday int
	switch {
	case dow == 6:
		daysUntilSaturday = 0
	case dow == 0:
		daysUntilSaturday = 6
	default:
		daysUntilSaturday = 6 - dow
	}

	nextSaturday := now.Add(time.Duration(daysUntilSaturday) * oneDay)
	return FormatKstYmd(nextSaturday)
}

// AddDaysToYmd adds days to a date string (YYYY-MM-DD or YYYYMMDD) and returns
// YYYY-MM-DD.
func AddDaysToYmd(dateStr string, days int) string {
	y, m, d := parseYmd(dateStr)
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC).AddDate(0, 0, days)
	return formatYmd(t.Year(), int(t.Month()), t.Day())
}

// AddYearsAndDays adds years (applied first, normalising like JS
// setUTCFullYear) and then days to a date string, returning YYYY-MM-DD.
func AddYearsAndDays(dateStr string, years, days int) string {
	y, m, d := parseYmd(dateStr)
	yearShifted := time.Date(y+years, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	t := yearShifted.AddDate(0, 0, days)
	return formatYmd(t.Year(), int(t.Month()), t.Day())
}

func normalizeYmd(dateStr string) string {
	if strings.Contains(dateStr, "-") {
		return dateStr
	}
	return dateStr[0:4] + "-" + dateStr[4:6] + "-" + dateStr[6:8]
}

func parseYmd(dateStr string) (year, month, day int) {
	parts := strings.Split(normalizeYmd(dateStr), "-")
	year, _ = strconv.Atoi(parts[0])
	month, _ = strconv.Atoi(parts[1])
	day, _ = strconv.Atoi(parts[2])
	return year, month, day
}

func formatYmd(year, month, day int) string {
	return pad4(year) + "-" + pad2(month) + "-" + pad2(day)
}

func pad2(n int) string { return fmt.Sprintf("%02d", n) }

func pad4(n int) string { return fmt.Sprintf("%04d", n) }
