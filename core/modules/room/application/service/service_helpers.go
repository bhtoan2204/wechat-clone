package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/stackErr"

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
	// actor, err := txRepos.RoomMemberReadRepository().GetRoomMemberByAccount(ctx, roomID, actorID)
	// if err != nil {
	// 	return nil, stackErr.Error(err)
	// }
	message, err := entity.NewSystemMessage(newUUID(), roomID, actorID, body, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := txRepos.MessageRepository().CreateMessage(ctx, message); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := txRepos.MessageReadRepository().UpsertMessage(ctx, message); err != nil {
		return nil, stackErr.Error(err)
	}
	return message, nil
}

func presenceKey(accountID string) string {
	return "chat:presence:" + strings.TrimSpace(accountID)
}

func currentActor(ctx context.Context) (*actorctx.Actor, bool) {
	return actorctx.FromContext(ctx)
}
