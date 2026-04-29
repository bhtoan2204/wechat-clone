package command

import (
	"context"
	"fmt"
	"net/http"
	"time"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/support"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/infra/storage"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type createPresignedURLHandler struct {
	storage storage.Storage
}

func NewCreatePresignedUrlHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.CreatePresignedUrlRequest, *out.CreatePresignedUrlResponse] {
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
