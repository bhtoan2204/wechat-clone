package utils

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"wechat-clone/core/shared/pkg/stackErr"
)

func EncodeCursor(createdAt string, id string) string {
	raw := createdAt + "|" + id
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(cursor string) (time.Time, string, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", stackErr.Error(err)
	}

	parts := strings.Split(string(data), "|")
	if len(parts) != 2 {
		return time.Time{}, "", stackErr.Error(errors.New("invalid cursor"))
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", stackErr.Error(err)
	}

	return t, parts[1], nil
}

func NullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func StringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
