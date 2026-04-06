package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/command"
	"go-socket/core/modules/account/application/query"
	accountservice "go-socket/core/modules/account/application/service"
	accountrepo "go-socket/core/modules/account/infra/persistent/repository"
	accountserver "go-socket/core/modules/account/transport/server"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	accountRepos := accountrepo.NewRepoImpl(appContext)
	accountAggregateService := accountservice.NewAccountAggregateService()
	emailVerificationService := accountservice.NewEmailVerificationService(appContext.GetCache(), appContext.GetSMTP())

	login := cqrs.NewDispatcher(command.NewLoginHandler(appContext, accountRepos))
	register := cqrs.NewDispatcher(command.NewRegisterHandler(appContext, accountRepos, accountAggregateService))
	logout := cqrs.NewDispatcher(command.NewLogoutHandler())
	getProfile := cqrs.NewDispatcher(query.NewGetProfileHandler(accountRepos))
	getAvatar := cqrs.NewDispatcher(query.NewGetAvatarHandler(accountRepos, appContext.GetStorage()))
	getPresignedUrl := cqrs.NewDispatcher(command.NewCreatePresignedUrlHandler(appContext))
	updateProfile := cqrs.NewDispatcher(command.NewUpdateProfileHandler(accountRepos, accountAggregateService))
	verifyEmail := cqrs.NewDispatcher(command.NewVerifyEmailHandler(accountRepos, accountAggregateService, emailVerificationService))
	confirmVerifyEmail := cqrs.NewDispatcher(command.NewConfirmVerifyEmailHandler(accountRepos, accountAggregateService, emailVerificationService))
	changePassword := cqrs.NewDispatcher(command.NewChangePasswordHandler(appContext, accountRepos, accountAggregateService))

	server, err := accountserver.NewHTTPServer(
		login,
		register,
		logout,
		getProfile,
		updateProfile,
		verifyEmail,
		confirmVerifyEmail,
		changePassword,
		getAvatar,
		getPresignedUrl,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
