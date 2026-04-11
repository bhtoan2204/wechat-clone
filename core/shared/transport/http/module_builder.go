package http

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/shared/pkg/stackErr"
)

// ModuleBuilder builds a module HTTP server.
type ModuleBuilder func(ctx context.Context, appCtx *appCtx.AppContext) (HTTPServer, error)

// BuildModuleServers builds module servers from the provided builders.
func BuildModuleServers(ctx context.Context, appCtx *appCtx.AppContext, builders ...ModuleBuilder) ([]HTTPServer, error) {
	servers := make([]HTTPServer, 0, len(builders))
	for idx, builder := range builders {
		if builder == nil {
			return nil, stackErr.Error(fmt.Errorf("module builder %d is nil", idx))
		}
		server, err := builder(ctx, appCtx)
		if err != nil {
			return nil, stackErr.Error(fmt.Errorf("build module server %d failed: %v", idx, err))
		}
		if server == nil {
			continue
		}
		servers = append(servers, server)
	}
	return servers, nil
}
