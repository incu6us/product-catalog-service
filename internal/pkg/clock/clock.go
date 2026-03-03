package clock

import "time"

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

// RealClock returns the actual current time.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// FixedClock returns a fixed time for testing.
type FixedClock struct {
	T time.Time
}

// Now returns the fixed time.
func (c FixedClock) Now() time.Time {
	return c.T.UTC()
}
