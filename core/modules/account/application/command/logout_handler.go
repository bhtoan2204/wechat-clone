package command

import (
	"context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
)

type logoutHandler struct{}

func NewLogoutHandler() cqrs.Handler[*in.LogoutRequest, *out.LogoutResponse] {
	return &logoutHandler{}
}

func (u *logoutHandler) Handle(ctx context.Context, req *in.LogoutRequest) (*out.LogoutResponse, error) {
	return &out.LogoutResponse{
		Message: "Logout successful",
	}, nil
}
