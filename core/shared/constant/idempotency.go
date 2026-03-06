package constant

import "time"

const (
	DEFAULT_IDEMPOTENCY_LOCK_TTL = time.Minute * 5
	DEFAULT_IDEMPOTENCY_DONE_TTL = time.Hour * 24
)
