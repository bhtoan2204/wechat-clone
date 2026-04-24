// CODE_GENERATOR: assembly
package assembly

import (
	"context"
	appCtx "wechat-clone/core/context"
	infrahttp "wechat-clone/core/shared/transport/http"
)

func BuildHTTPServer(ctx context.Context, appContext *appCtx.AppContext) (infrahttp.HTTPServer, error) {
	return buildHTTPServer(ctx, appContext)
}
