package types

import (
	"database/sql/driver"
	"strings"
)

type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
	RoomTypeDirect  RoomType = "direct"
	RoomTypeGroup   RoomType = "group"
)

func (r RoomType) Normalize() RoomType {
	return RoomType(strings.ToLower(strings.TrimSpace(string(r))))
}

func (r RoomType) IsValid() bool {
	switch r.Normalize() {
	case RoomTypePublic, RoomTypePrivate, RoomTypeDirect, RoomTypeGroup:
		return true
	default:
		return false
	}
}

func (r RoomType) Value() (driver.Value, error) {
	return string(r), nil
}

func (r *RoomType) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}

	*r = RoomType(value.(string))
	return nil
}

func (r *RoomType) String() string {
	return string(*r)
}
