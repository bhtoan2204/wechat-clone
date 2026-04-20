// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"strings"
	"wechat-clone/core/shared/pkg/stackErr"
)

type CreateChatMessagePresignedURLRequest struct {
	RoomID      string `json:"room_id" form:"room_id" binding:"required"`
	MessageType string `json:"message_type" form:"message_type" binding:"required"`
	FileName    string `json:"file_name" form:"file_name" binding:"required"`
	MimeType    string `json:"mime_type" form:"mime_type"`
}

func (r *CreateChatMessagePresignedURLRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.MessageType = strings.TrimSpace(r.MessageType)
	r.FileName = strings.TrimSpace(r.FileName)
	r.MimeType = strings.TrimSpace(r.MimeType)
}

func (r *CreateChatMessagePresignedURLRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	if r.MessageType == "" {
		return stackErr.Error(errors.New("message_type is required"))
	}
	if r.FileName == "" {
		return stackErr.Error(errors.New("file_name is required"))
	}
	return nil
}
