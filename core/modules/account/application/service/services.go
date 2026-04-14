package service

import (
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/domain/repos"
)

//go:generate mockgen -package=service -destination=services_mock.go -source=services.go
type Services interface {
	AuthenticationService() AuthenticationService
	EmailVerificationService() EmailVerificationService
}

type services struct {
	authenticationService    AuthenticationService
	emailVerificationService EmailVerificationService
}

func NewServices(appCtx *appCtx.AppContext, repos repos.Repos) Services {
	return &services{
		authenticationService:    NewAuthenticationService(appCtx, repos),
		emailVerificationService: NewEmailVerificationService(appCtx),
	}
}

func (s *services) AuthenticationService() AuthenticationService {
	return s.authenticationService
}

func (s *services) EmailVerificationService() EmailVerificationService {
	return s.emailVerificationService
}
