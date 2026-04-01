package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/command"
	"go-socket/core/modules/account/application/query"
	accountrepo "go-socket/core/modules/account/infra/persistent/repository"
	accountserver "go-socket/core/modules/account/transport/server"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	accountRepos := accountrepo.NewRepoImpl(appContext)

	login := cqrs.NewDispatcher(command.NewLoginHandler(appContext, accountRepos))
	register := cqrs.NewDispatcher(command.NewRegisterHandler(appContext, accountRepos))
	logout := cqrs.NewDispatcher(command.NewLogoutHandler())
	getProfile := cqrs.NewDispatcher(query.NewGetProfileHandler(accountRepos))

	server, err := accountserver.NewServer(login, register, logout, getProfile)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
