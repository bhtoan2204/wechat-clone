package in

import (
	"errors"
	"strings"
)

type CreateGroupChatRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	MemberIDs   []string `json:"member_ids"`
}

func (r *CreateGroupChatRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	for i := range r.MemberIDs {
		r.MemberIDs[i] = strings.TrimSpace(r.MemberIDs[i])
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
