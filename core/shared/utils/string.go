package utils

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

func EncodeCursor(createdAt string, id string) string {
	raw := createdAt + "|" + id
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(cursor string) (time.Time, string, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", err
	}

	parts := strings.Split(string(data), "|")
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("invalid cursor")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}

	return t, parts[1], nil
}
