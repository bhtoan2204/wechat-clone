package utils

func BoolToSmallInt(value bool) int16 {
	if value {
		return 1
	}
	return 0
}
