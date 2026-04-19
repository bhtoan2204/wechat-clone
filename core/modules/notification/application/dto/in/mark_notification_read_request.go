package in

import (
	"errors"
	"strings"

	"wechat-clone/core/shared/pkg/stackErr"
)

type MarkNotificationReadRequest struct {
	NotificationID string `json:"notification_id" form:"notification_id"`
}

func (r *MarkNotificationReadRequest) Normalize() {
	r.NotificationID = strings.TrimSpace(r.NotificationID)
}

func (r *MarkNotificationReadRequest) Validate() error {
	r.Normalize()
	if r.NotificationID == "" {
		return stackErr.Error(errors.New("notification_id is required"))
	}
	return nil
}
