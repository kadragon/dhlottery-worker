package datekst

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return parsed
}

func TestCalculatePreviousWeekRange(t *testing.T) {
	// Monday 10:00 KST -> previous week Mon 2025-12-08 .. Sun 2025-12-14.
	r := CalculatePreviousWeekRange(mustTime(t, "2025-12-15T10:00:00+09:00"))
	if r.StartDate != "2025-12-08" {
		t.Errorf("StartDate = %q, want 2025-12-08", r.StartDate)
	}
	if r.EndDate != "2025-12-14" {
		t.Errorf("EndDate = %q, want 2025-12-14", r.EndDate)
	}
	if got := r.End.Sub(r.Start) + time.Millisecond; got != 7*24*time.Hour {
		t.Errorf("range span = %v, want 7 days", got)
	}
}

func TestNextSaturdayKst(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"monday", "2025-12-15T10:00:00+09:00", "2025-12-20"},
		{"saturday today", "2025-12-20T10:00:00+09:00", "2025-12-20"},
		{"sunday", "2025-12-21T10:00:00+09:00", "2025-12-27"},
		{"late saturday KST", "2025-12-20T18:00:00+09:00", "2025-12-20"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextSaturdayKst(mustTime(t, c.in)); got != c.want {
				t.Errorf("NextSaturdayKst = %q, want %q", got, c.want)
			}
		})
	}
}

func TestAddDaysToYmd(t *testing.T) {
	cases := []struct {
		in   string
		days int
		want string
	}{
		{"2025-12-28", 7, "2026-01-04"},
		{"20251228", 7, "2026-01-04"},
		{"2025-01-30", 2, "2025-02-01"},
		{"2026-02-19", 7, "2026-02-26"},
	}
	for _, c := range cases {
		if got := AddDaysToYmd(c.in, c.days); got != c.want {
			t.Errorf("AddDaysToYmd(%q,%d) = %q, want %q", c.in, c.days, got, c.want)
		}
	}
}

func TestAddYearsAndDays(t *testing.T) {
	cases := []struct {
		in    string
		years int
		days  int
		want  string
	}{
		{"2025-12-20", 1, 1, "2026-12-21"},
		{"2024-02-29", 1, 0, "2025-03-01"},
	}
	for _, c := range cases {
		if got := AddYearsAndDays(c.in, c.years, c.days); got != c.want {
			t.Errorf("AddYearsAndDays(%q,%d,%d) = %q, want %q", c.in, c.years, c.days, got, c.want)
		}
	}
}

func TestFormatKstYmd(t *testing.T) {
	// 2025-12-20T18:00 KST = 2025-12-20T09:00Z; in KST the date is 2025-12-20.
	if got := FormatKstYmd(mustTime(t, "2025-12-20T18:00:00+09:00")); got != "2025-12-20" {
		t.Errorf("FormatKstYmd = %q, want 2025-12-20", got)
	}
}
