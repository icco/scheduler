package scheduler

import "time"

type Entry struct {
	start time.Time
	end   time.Time
	user  string
}
