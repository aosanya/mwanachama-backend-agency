package mwanachamaagency

import "time"

func SetRowClock(now func() time.Time) func() {
	previous := rowClock
	rowClock = now
	return func() { rowClock = previous }
}
