package query

import (
	"context"
	"net/url"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/projection"
	"wechat-clone/core/shared/infra/storage"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

const avatarPresignedURLTTL = 15 * time.Minute

type getAvatarHandler struct {
	accountReadRepo projection.AccountReadRepository
	storage         storage.Storage
}

func NewGetAvatarHandler(appCtx *appCtx.AppContext, accountReadRepo projection.AccountReadRepository) cqrs.Handler[*in.GetAvatarRequest, *out.GetAvatarResponse] {
	return &getAvatarHandler{
		accountReadRepo: accountReadRepo,
		storage:         appCtx.GetStorage(),
	}
}

func (u *getAvatarHandler) Handle(ctx context.Context, req *in.GetAvatarRequest) (*out.GetAvatarResponse, error) {
	log := logging.FromContext(ctx).Named("GetAvatar")

	accountEntity, err := u.accountReadRepo.GetAccountByID(ctx, req.AccountID)
	if err != nil {
		log.Errorw("Failed to get account by ID", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if accountEntity.AvatarObjectKey == nil || *accountEntity.AvatarObjectKey == "" {
		return &out.GetAvatarResponse{}, nil
	}

	isAbsoluteURL := func(value string) bool {
		parsed, err := url.ParseRequestURI(value)
		if err != nil {
			return false
		}

		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}

	avatarValue := *accountEntity.AvatarObjectKey
	if isAbsoluteURL(avatarValue) {
		return &out.GetAvatarResponse{
			URL: avatarValue,
		}, nil
	}

	url, err := u.storage.PresignedGetObjectURL(ctx, *accountEntity.AvatarObjectKey, avatarPresignedURLTTL)
	if err != nil {
		log.Errorw("Failed to generate presigned avatar URL", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	return &out.GetAvatarResponse{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(avatarPresignedURLTTL).Format(time.RFC3339),
	}, nil
}
