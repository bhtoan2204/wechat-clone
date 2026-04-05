package in

import (
	"errors"
	"strings"
)

type CreateDirectConversationRequest struct {
	PeerAccountID string `json:"peer_account_id"`
}

func (r *CreateDirectConversationRequest) Validate() error {
	r.PeerAccountID = strings.TrimSpace(r.PeerAccountID)
	if r.PeerAccountID == "" {
		return errors.New("peer_account_id is required")
	}
	return nil
}
