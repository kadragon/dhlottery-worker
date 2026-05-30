package format

import "testing"

func TestKoreanNumber(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1,000"},
		{5000, "5,000"},
		{9000, "9,000"},
		{20000, "20,000"},
		{1234567, "1,234,567"},
		{5000000, "5,000,000"},
		{2000000000, "2,000,000,000"},
		{100000, "100,000"},
		{-1000, "-1,000"},
		{-100, "-100"},
	}
	for _, c := range cases {
		if got := KoreanNumber(c.in); got != c.want {
			t.Errorf("KoreanNumber(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCurrency(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{4000, "4,000원"},
		{5000, "5,000원"},
		{10000, "10,000원"},
		{50000, "50,000원"},
	}
	for _, c := range cases {
		if got := Currency(c.in); got != c.want {
			t.Errorf("Currency(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
