package in

import (
	"errors"
	"strings"
)

type GetChatPresenceRequest struct {
	AccountID string `json:"account_id" uri:"account_id"`
}

func (r *GetChatPresenceRequest) Validate() error {
	r.AccountID = strings.TrimSpace(r.AccountID)
	if r.AccountID == "" {
		return errors.New("account_id is required")
	}
	return nil
}
