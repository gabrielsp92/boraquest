// Package clock provides time sources.
package clock

import "time"

// SystemClock returns the real wall-clock time in UTC.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
