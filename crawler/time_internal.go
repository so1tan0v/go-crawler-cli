package crawler

import "time"

var timeNow = time.Now

func nowUTC() time.Time {
	return timeNow().UTC().Truncate(time.Second)
}
