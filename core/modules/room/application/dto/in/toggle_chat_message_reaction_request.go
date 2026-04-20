// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"strings"
	"wechat-clone/core/shared/pkg/stackErr"
)

type ToggleChatMessageReactionRequest struct {
	MessageID string `json:"message_id" form:"message_id" binding:"required"`
	Emoji     string `json:"emoji" form:"emoji" binding:"required"`
}

func (r *ToggleChatMessageReactionRequest) Normalize() {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Emoji = strings.TrimSpace(r.Emoji)
}

func (r *ToggleChatMessageReactionRequest) Validate() error {
	r.Normalize()
	if r.MessageID == "" {
		return stackErr.Error(errors.New("message_id is required"))
	}
	if r.Emoji == "" {
		return stackErr.Error(errors.New("emoji is required"))
	}
	return nil
}
