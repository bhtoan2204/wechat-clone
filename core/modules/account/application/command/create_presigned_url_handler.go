package command

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/support"
	"go-socket/core/shared/infra/storage"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
	"net/http"
	"time"
)

type createPresignedURLHandler struct {
	storage storage.Storage

	avatarBucket string
}

func NewCreatePresignedUrlHandler(appCtx *appCtx.AppContext) cqrs.Handler[*in.CreatePresignedUrlRequest, *out.CreatePresignedUrlResponse] {
	return &createPresignedURLHandler{
		storage: appCtx.GetStorage(),
	}
}

func (u *createPresignedURLHandler) Handle(ctx context.Context, req *in.CreatePresignedUrlRequest) (*out.CreatePresignedUrlResponse, error) {
	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	putPresignedUrl, expires, err := u.storage.PresignedPutObjectURL(ctx, fmt.Sprintf("avatar/%s", accountID), time.Minute*15)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &out.CreatePresignedUrlResponse{
		PresignedURL: putPresignedUrl,
		ExpiresAt:    expires.Format(time.RFC3339),
		Method:       http.MethodPut,
	}, nil
}
