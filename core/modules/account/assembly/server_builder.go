package assembly

import (
	appCtx "go-socket/core/context"
	accountserver "go-socket/core/modules/account/transport/server"
	stackerr "go-socket/core/shared/pkg/stackErr"
)

func BuildServer(appContext *appCtx.AppContext) (accountserver.Server, error) {
	buses := BuildBuses(appContext)

	server, err := accountserver.NewServer(buses.Command, buses.Query)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
