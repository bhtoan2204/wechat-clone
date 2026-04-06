// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type CreateGroupChatRequest struct {
	Name        string   `json:"name" form:"name" binding:"required"`
	Description string   `json:"description" form:"description"`
	MemberIDs   []string `json:"member_ids" form:"member_ids"`
}

func (r *CreateGroupChatRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	for i := range r.MemberIDs {
		r.MemberIDs[i] = strings.TrimSpace(r.MemberIDs[i])
	}
}

func (r *CreateGroupChatRequest) Validate() error {
	r.Normalize()
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
