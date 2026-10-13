// Package weekresult holds the frozen outcome of a closed guild week.
package weekresult

import "time"

// Standing is one member's frozen row in a closed week. Rank starts at 1.
type Standing struct {
	MemberID  string
	Rank      int
	Points    int
	Completed int
}

// WeekResult is a closed week (Monday..Sunday) of a guild: its standings, sorted
// by rank, and a snapshot of the week prize that was in effect when it ended.
type WeekResult struct {
	GuildID   string
	WeekStart time.Time // date-only, a Monday
	WeekEnd   time.Time // date-only, the following Sunday
	Prize     string
	ClosedAt  time.Time
	Standings []Standing
}

// Winner returns the rank-1 standing, or false if the week has no standings.
func (w WeekResult) Winner() (Standing, bool) {
	if len(w.Standings) == 0 {
		return Standing{}, false
	}
	return w.Standings[0], true
}
