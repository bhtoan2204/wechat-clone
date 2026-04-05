package repository

func limitOrDefault(limit, defaultLimit, maxLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func int64Ptr(value int64) *int64 {
	return &value
}
