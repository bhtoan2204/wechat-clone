// CODE_GENERATOR - do not edit: request

package in

import (
	"strings"
)

type ListNotificationRequest struct {
	Cursor string `json:"cursor" form:"cursor"`
	Limit  int    `json:"limit" form:"limit"`
}

func (r *ListNotificationRequest) Normalize() {
	r.Cursor = strings.TrimSpace(r.Cursor)
}

func (r *ListNotificationRequest) Validate() error {
	r.Normalize()
	return nil
}
