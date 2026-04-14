// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type SendChatMessageRequest struct {
	RoomID                 string                          `json:"room_id" form:"room_id" binding:"required"`
	Message                string                          `json:"message" form:"message"`
	MessageType            string                          `json:"message_type" form:"message_type"`
	Mentions               []SendChatMessageMentionRequest `json:"mentions" form:"mentions"`
	MentionAll             bool                            `json:"mention_all" form:"mention_all"`
	ReplyToMessageID       string                          `json:"reply_to_message_id" form:"reply_to_message_id"`
	ForwardedFromMessageID string                          `json:"forwarded_from_message_id" form:"forwarded_from_message_id"`
	FileName               string                          `json:"file_name" form:"file_name"`
	FileSize               int64                           `json:"file_size" form:"file_size"`
	MimeType               string                          `json:"mime_type" form:"mime_type"`
	ObjectKey              string                          `json:"object_key" form:"object_key"`
}

type SendChatMessageMentionRequest struct {
	AccountID string `json:"account_id" form:"account_id"`
}

func (r *SendChatMessageMentionRequest) Normalize() {
	r.AccountID = strings.TrimSpace(r.AccountID)
}

func (r *SendChatMessageRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.Message = strings.TrimSpace(r.Message)
	r.MessageType = strings.TrimSpace(r.MessageType)
	for idx := range r.Mentions {
		r.Mentions[idx].Normalize()
	}
	r.ReplyToMessageID = strings.TrimSpace(r.ReplyToMessageID)
	r.ForwardedFromMessageID = strings.TrimSpace(r.ForwardedFromMessageID)
	r.FileName = strings.TrimSpace(r.FileName)
	r.MimeType = strings.TrimSpace(r.MimeType)
	r.ObjectKey = strings.TrimSpace(r.ObjectKey)
}

func (r *SendChatMessageRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	return nil
}
