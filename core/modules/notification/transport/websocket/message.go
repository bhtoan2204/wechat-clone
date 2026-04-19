package socket

import "encoding/json"

type Message struct {
	RoomID string          `json:"room_id"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data,omitempty"`
}
