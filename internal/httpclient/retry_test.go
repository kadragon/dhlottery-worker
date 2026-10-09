package httpclient

import (
	"reflect"
	"testing"
	"time"
)

func TestRetryStopsOnSuccess(t *testing.T) {
	var slept []time.Duration
	var calls []bool
	Retry([]time.Duration{time.Second, 2 * time.Second}, func(d time.Duration) { slept = append(slept, d) },
		func(_ int, final bool) (bool, time.Duration) {
			calls = append(calls, final)
			return len(calls) < 2, 0
		})
	if want := []bool{false, false}; !reflect.DeepEqual(calls, want) {
		t.Errorf("final flags = %v, want %v", calls, want)
	}
	if want := []time.Duration{time.Second}; !reflect.DeepEqual(slept, want) {
		t.Errorf("slept = %v, want %v", slept, want)
	}
}

func TestRetryExhaustsDelaysAndMarksFinal(t *testing.T) {
	var slept []time.Duration
	var calls []bool
	Retry([]time.Duration{time.Second, 2 * time.Second}, func(d time.Duration) { slept = append(slept, d) },
		func(_ int, final bool) (bool, time.Duration) {
			calls = append(calls, final)
			return true, 0
		})
	if want := []bool{false, false, true}; !reflect.DeepEqual(calls, want) {
		t.Errorf("final flags = %v, want %v", calls, want)
	}
	if want := []time.Duration{time.Second, 2 * time.Second}; !reflect.DeepEqual(slept, want) {
		t.Errorf("slept = %v, want %v", slept, want)
	}
}

func TestRetryWaitOverridesDelay(t *testing.T) {
	var slept []time.Duration
	n := 0
	Retry([]time.Duration{time.Second}, func(d time.Duration) { slept = append(slept, d) },
		func(int, bool) (bool, time.Duration) {
			n++
			return n == 1, 7 * time.Second
		})
	if want := []time.Duration{7 * time.Second}; !reflect.DeepEqual(slept, want) {
		t.Errorf("slept = %v, want %v", slept, want)
	}
}

func TestRetryPassesAttemptIndex(t *testing.T) {
	var idx []int
	Retry([]time.Duration{time.Second, 2 * time.Second}, func(time.Duration) {},
		func(i int, _ bool) (bool, time.Duration) {
			idx = append(idx, i)
			return true, 0
		})
	if want := []int{0, 1, 2}; !reflect.DeepEqual(idx, want) {
		t.Errorf("attempt indices = %v, want %v", idx, want)
	}
}
