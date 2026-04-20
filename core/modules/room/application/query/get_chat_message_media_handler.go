package query

import (
	"context"
	"strings"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	roomsupport "wechat-clone/core/modules/room/application/support"
	"wechat-clone/core/modules/room/domain/entity"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/shared/infra/storage"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

const chatMessageMediaURLTTL = 15 * time.Minute

type getChatMessageMediaHandler struct {
	baseRepo roomrepos.Repos
	storage  storage.Storage
}

func NewGetChatMessageMediaHandler(appCtx *appCtx.AppContext, baseRepo roomrepos.Repos) cqrs.Handler[*in.GetChatMessageMediaRequest, *out.GetChatMessageMediaResponse] {
	return &getChatMessageMediaHandler{
		baseRepo: baseRepo,
		storage:  appCtx.GetStorage(),
	}
}

func (h *getChatMessageMediaHandler) Handle(ctx context.Context, req *in.GetChatMessageMediaRequest) (*out.GetChatMessageMediaResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	roomID := strings.TrimSpace(req.RoomID)
	objectKey := strings.TrimSpace(req.ObjectKey)

	if !strings.HasPrefix(objectKey, roomMediaPrefix(roomID)) {
		return nil, stackErr.Error(entity.ErrRoomMemberRequired)
	}

	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if !isRoomMember(agg.Members(), accountID) {
		return nil, stackErr.Error(entity.ErrRoomMemberRequired)
	}

	url, err := h.storage.PresignedGetObjectURL(ctx, objectKey, chatMessageMediaURLTTL)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.GetChatMessageMediaResponse{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(chatMessageMediaURLTTL).Format(time.RFC3339),
	}, nil
}

func roomMediaPrefix(roomID string) string {
	return "chat/" + strings.TrimSpace(roomID) + "/"
}

func isRoomMember(members []*entity.RoomMemberEntity, accountID string) bool {
	accountID = strings.TrimSpace(accountID)
	for _, member := range members {
		if member == nil {
			continue
		}
		if strings.TrimSpace(member.AccountID) == accountID {
			return true
		}
	}
	return false
}
