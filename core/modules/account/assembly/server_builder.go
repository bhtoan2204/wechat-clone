package assembly

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/command"
	"wechat-clone/core/modules/account/application/provider"
	"wechat-clone/core/modules/account/application/provider/google"
	"wechat-clone/core/modules/account/application/query"
	accountrepo "wechat-clone/core/modules/account/infra/persistent/repository"
	accountes "wechat-clone/core/modules/account/infra/projection/elasticsearch"
	accountserver "wechat-clone/core/modules/account/transport/server"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/transport/http"
)

func buildHTTPServer(ctx context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	searchRepository, err := accountes.NewAccountSearchRepository(appContext.GetConfig().ElasticsearchConfig, appContext.GetElasticsearchClient())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	accountRepos := accountrepo.NewRepoImpl(appContext.GetDB(), appContext.GetCache())
	accountReadRepo := accountrepo.NewAccountRepoImpl(appContext.GetDB(), appContext.GetCache(), true, nil, searchRepository)
	authProviderRegistry := provider.NewProviderRegistry()
	authProviderRegistry.Register(google.NewGoogleProvider(ctx, appContext.GetConfig()))

	login := cqrs.NewDispatcher(command.NewLoginHandler(appContext, accountRepos))
	register := cqrs.NewDispatcher(command.NewRegisterHandler(appContext, accountRepos))
	logout := cqrs.NewDispatcher(command.NewLogoutHandler(appContext, accountRepos))
	getProfile := cqrs.NewDispatcher(query.NewGetProfileHandler(appContext, accountReadRepo))
	getAvatar := cqrs.NewDispatcher(query.NewGetAvatarHandler(appContext, accountReadRepo))
	getPresignedUrl := cqrs.NewDispatcher(command.NewCreatePresignedUrlHandler(appContext, accountRepos))
	updateProfile := cqrs.NewDispatcher(command.NewUpdateProfileHandler(appContext, accountRepos))
	verifyEmail := cqrs.NewDispatcher(command.NewVerifyEmailHandler(appContext, accountRepos))
	confirmVerifyEmail := cqrs.NewDispatcher(command.NewConfirmVerifyEmailHandler(appContext, accountRepos))
	changePassword := cqrs.NewDispatcher(command.NewChangePasswordHandler(appContext, accountRepos))
	searchUsers := cqrs.NewDispatcher(query.NewSearchUsers(appContext, accountReadRepo))
	refresh := cqrs.NewDispatcher(command.NewRefresh(appContext, accountRepos))
	loginGoogle := cqrs.NewDispatcher(command.NewLoginGoogle(appContext, accountRepos, authProviderRegistry))
	callbackGoogle := cqrs.NewDispatcher(command.NewCallbackGoogle(appContext, accountRepos, authProviderRegistry))
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
		loginGoogle,
		callbackGoogle,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
