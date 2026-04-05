package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/infra/xpaseto"

	"gorm.io/gorm"
)

func requireRoomRole(ctx context.Context, roomRepo repos.RoomRepository, memberRepo repos.RoomMemberRepository, roomID, accountID string) (*entity.RoomMemberEntity, *entity.Room, error) {
	room, err := roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}

	member, err := memberRepo.GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("account is not a member of this room")
		}
		return nil, nil, err
	}

	return member, room, nil
}

func createSystemMessageTx(ctx context.Context, txRepos repos.Repos, roomID, actorID, body string, now time.Time) (*entity.MessageEntity, error) {
	message := &entity.MessageEntity{
		ID:          newUUID(),
		RoomID:      roomID,
		SenderID:    actorID,
		Message:     body,
		MessageType: "system",
		CreatedAt:   now,
	}
	if err := txRepos.MessageRepository().CreateMessage(ctx, message); err != nil {
		return nil, err
	}
	if err := txRepos.MessageReadRepository().UpsertMessage(ctx, message); err != nil {
		return nil, err
	}
	return message, nil
}

func canonicalDirectKey(a, b string) string {
	ids := []string{strings.TrimSpace(a), strings.TrimSpace(b)}
	if ids[0] > ids[1] {
		ids[0], ids[1] = ids[1], ids[0]
	}
	return strings.Join(ids, ":")
}

func normalizeMessageType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "text":
		return "text"
	case "system":
		return "system"
	case "image":
		return "image"
	case "file":
		return "file"
	default:
		return ""
	}
}

func presenceKey(accountID string) string {
	return "chat:presence:" + strings.TrimSpace(accountID)
}

func currentAccountPayload(ctx context.Context) (*xpaseto.PasetoPayload, bool) {
	payload, ok := ctx.Value("account").(*xpaseto.PasetoPayload)
	if !ok || payload == nil {
		return nil, false
	}
	return payload, true
}
