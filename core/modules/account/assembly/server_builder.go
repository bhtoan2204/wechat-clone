package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	accountserver "go-socket/core/modules/account/transport/server"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	buses := BuildBuses(appContext)

	server, err := accountserver.NewServer(buses.Command, buses.Query)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
