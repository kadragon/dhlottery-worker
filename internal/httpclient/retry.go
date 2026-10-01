package httpclient

import "time"

// Retry calls attempt up to len(delays)+1 times. final is true on the last
// permitted call, so attempt can log a retryable failure as a warning or as
// the terminal error. attempt returns again=true to request another call; a
// positive wait replaces the scheduled delay (e.g. a server's Retry-After).
// sleep is the caller's injectable seam, so tests never really wait.
func Retry(delays []time.Duration, sleep func(time.Duration), attempt func(final bool) (again bool, wait time.Duration)) {
	for i := 0; ; i++ {
		again, wait := attempt(i == len(delays))
		if !again || i == len(delays) {
			return
		}
		if wait <= 0 {
			wait = delays[i]
		}
		sleep(wait)
	}
}
