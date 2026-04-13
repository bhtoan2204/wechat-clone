package projection

import "time"

type MessageListOptions struct {
	Limit     int
	BeforeID  string
	BeforeAt  *time.Time
	Ascending bool
}
