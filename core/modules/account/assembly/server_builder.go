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

func buildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	accountRepos := accountrepo.NewRepoImpl(appContext.GetDB(), appContext.GetCache())
	accountServices := accountservice.NewServices(appContext, accountRepos)

	login := cqrs.NewDispatcher(command.NewLoginHandler(appContext, accountRepos, accountServices))
	register := cqrs.NewDispatcher(command.NewRegisterHandler(appContext, accountRepos, accountServices))
	logout := cqrs.NewDispatcher(command.NewLogoutHandler(appContext, accountRepos, accountServices))
	getProfile := cqrs.NewDispatcher(query.NewGetProfileHandler(appContext, accountRepos, accountServices))
	getAvatar := cqrs.NewDispatcher(query.NewGetAvatarHandler(appContext, accountRepos, accountServices))
	getPresignedUrl := cqrs.NewDispatcher(command.NewCreatePresignedUrlHandler(appContext, accountRepos, accountServices))
	updateProfile := cqrs.NewDispatcher(command.NewUpdateProfileHandler(appContext, accountRepos, accountServices))
	verifyEmail := cqrs.NewDispatcher(command.NewVerifyEmailHandler(appContext, accountRepos, accountServices))
	confirmVerifyEmail := cqrs.NewDispatcher(command.NewConfirmVerifyEmailHandler(appContext, accountRepos, accountServices))
	changePassword := cqrs.NewDispatcher(command.NewChangePasswordHandler(appContext, accountRepos, accountServices))
	searchUsers := cqrs.NewDispatcher(query.NewSearchUsers(appContext, accountRepos, accountServices))
	refresh := cqrs.NewDispatcher(command.NewRefresh(appContext, accountRepos, accountServices))
	server, err := accountserver.NewHTTPServer(
		login,
		register,
		logout,
		refresh,
		getProfile,
		updateProfile,
		verifyEmail,
		confirmVerifyEmail,
		changePassword,
		getAvatar,
		getPresignedUrl,
		searchUsers,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
