// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type SavePushSubscriptionRequest struct {
	Endpoint string            `json:"endpoint" form:"endpoint" binding:"required"`
	Keys     map[string]string `json:"keys" form:"keys" binding:"required"`
}

func (r *SavePushSubscriptionRequest) Normalize() {
	r.Endpoint = strings.TrimSpace(r.Endpoint)
	for key, value := range r.Keys {
		r.Keys[key] = strings.TrimSpace(value)
	}
}

func (r *SavePushSubscriptionRequest) Validate() error {
	r.Normalize()
	if r.Endpoint == "" {
		return stackErr.Error(errors.New("endpoint is required"))
	}
	if len(r.Keys) == 0 {
		return stackErr.Error(errors.New("keys is required"))
	}
	return nil
}
