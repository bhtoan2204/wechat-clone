package in

import (
	"errors"
	"strings"
)

type SendChatMessageRequest struct {
	RoomID                 string `json:"room_id"`
	Message                string `json:"message"`
	MessageType            string `json:"message_type"`
	ReplyToMessageID       string `json:"reply_to_message_id"`
	ForwardedFromMessageID string `json:"forwarded_from_message_id"`
	FileName               string `json:"file_name"`
	FileSize               int64  `json:"file_size"`
	MimeType               string `json:"mime_type"`
	ObjectKey              string `json:"object_key"`
}

func (r *SendChatMessageRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.Message = strings.TrimSpace(r.Message)
	r.MessageType = strings.TrimSpace(r.MessageType)
	r.ReplyToMessageID = strings.TrimSpace(r.ReplyToMessageID)
	r.ForwardedFromMessageID = strings.TrimSpace(r.ForwardedFromMessageID)
	r.FileName = strings.TrimSpace(r.FileName)
	r.MimeType = strings.TrimSpace(r.MimeType)
	r.ObjectKey = strings.TrimSpace(r.ObjectKey)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	return nil
}
