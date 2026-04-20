package command

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
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

	"github.com/google/uuid"
)

const chatMessageUploadURLTTL = 15 * time.Minute

type createChatMessagePresignedURLHandler struct {
	baseRepo roomrepos.Repos
	storage  storage.Storage
}

func NewCreateChatMessagePresignedURLHandler(appCtx *appCtx.AppContext, baseRepo roomrepos.Repos) cqrs.Handler[*in.CreateChatMessagePresignedURLRequest, *out.CreateChatMessagePresignedURLResponse] {
	return &createChatMessagePresignedURLHandler{
		baseRepo: baseRepo,
		storage:  appCtx.GetStorage(),
	}
}

func (h *createChatMessagePresignedURLHandler) Handle(ctx context.Context, req *in.CreateChatMessagePresignedURLRequest) (*out.CreateChatMessagePresignedURLResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	messageType, err := normalizeUploadMessageType(req.MessageType)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if !isRoomMember(agg.Members(), accountID) {
		return nil, stackErr.Error(entity.ErrRoomMemberRequired)
	}

	objectKey := buildChatMessageObjectKey(req.RoomID, accountID, messageType, req.FileName)
	putPresignedURL, expiresAt, err := h.storage.PresignedPutObjectURL(ctx, objectKey, chatMessageUploadURLTTL)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.CreateChatMessagePresignedURLResponse{
		PresignedURL: putPresignedURL,
		ObjectKey:    objectKey,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		Method:       http.MethodPut,
	}, nil
}

func normalizeUploadMessageType(messageType string) (string, error) {
	switch entity.NormalizeMessageType(messageType) {
	case entity.MessageTypeImage, entity.MessageTypeFile, entity.MessageTypeSticker:
		return entity.NormalizeMessageType(messageType), nil
	default:
		return "", stackErr.Error(entity.ErrMessageTypeInvalid)
	}
}

func buildChatMessageObjectKey(roomID, accountID, messageType, fileName string) string {
	extension := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	objectID := uuid.NewString()
	if extension == "" {
		return fmt.Sprintf("chat/%s/%s/%s/%s", strings.TrimSpace(roomID), strings.TrimSpace(accountID), strings.TrimSpace(messageType), objectID)
	}
	return fmt.Sprintf("chat/%s/%s/%s/%s%s", strings.TrimSpace(roomID), strings.TrimSpace(accountID), strings.TrimSpace(messageType), objectID, extension)
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
