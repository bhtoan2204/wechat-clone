package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/domain/aggregate"
	"wechat-clone/core/modules/room/domain/entity"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	roomtypes "wechat-clone/core/modules/room/types"
	sharedcache "wechat-clone/core/shared/infra/cache"
	"wechat-clone/core/shared/infra/lock"

	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"
)

func TestVideoCallServiceStartCallCreatesActiveSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repos := roomrepos.NewMockRepos(ctrl)
	roomAggRepo := roomrepos.NewMockRoomAggregateRepository(ctrl)
	cache := sharedcache.NewMockCache(ctrl)
	locker := lock.NewMockLock(ctrl)

	service := &videoCallService{
		baseRepo: repos,
		locker:   locker,
		store:    newVideoCallSessionCacheStore(cache),
	}

	roomAgg := testRoomAggregate(t, "room-1", "actor-1")

	locker.EXPECT().AcquireLock(gomock.Any(), "room:video_call:room-1", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	locker.EXPECT().ReleaseLock(gomock.Any(), "room:video_call:room-1", gomock.Any()).Return(true, nil)
	repos.EXPECT().RoomAggregateRepository().Return(roomAggRepo).AnyTimes()
	roomAggRepo.EXPECT().Load(gomock.Any(), "room-1").Return(roomAgg, nil).Times(1)
	cache.EXPECT().Get(gomock.Any(), "room:video_call:room-1").Return(nil, redisNilWrapped()).Times(1)
	cache.EXPECT().SetObject(gomock.Any(), "room:video_call:room-1", gomock.AssignableToTypeOf(&entity.VideoCallSession{}), gomock.Any()).Return(nil).Times(1)

	result, err := service.StartCall(context.Background(), apptypes.StartVideoCallCommand{
		RoomID:  "room-1",
		ActorID: "actor-1",
	})
	if err != nil {
		t.Fatalf("StartCall() error = %v", err)
	}
	if result == nil {
		t.Fatal("StartCall() result = nil")
	}
	if result.Status != entity.VideoCallStatusActive {
		t.Fatalf("Status = %s, want %s", result.Status, entity.VideoCallStatusActive)
	}
	if len(result.ParticipantAccountIDs) != 1 || result.ParticipantAccountIDs[0] != "actor-1" {
		t.Fatalf("participants = %#v, want [actor-1]", result.ParticipantAccountIDs)
	}
}

func TestVideoCallServiceRelaySignalRequiresParticipant(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repos := roomrepos.NewMockRepos(ctrl)
	roomAggRepo := roomrepos.NewMockRoomAggregateRepository(ctrl)
	cache := sharedcache.NewMockCache(ctrl)

	service := &videoCallService{
		baseRepo: repos,
		store:    newVideoCallSessionCacheStore(cache),
	}

	roomAgg := testRoomAggregate(t, "room-1", "actor-1")
	session := &entity.VideoCallSession{
		SessionID:             "session-1",
		RoomID:                "room-1",
		Status:                entity.VideoCallStatusActive,
		StartedByAccountID:    "other-user",
		ParticipantAccountIDs: []string{"other-user"},
		StartedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}

	repos.EXPECT().RoomAggregateRepository().Return(roomAggRepo).AnyTimes()
	roomAggRepo.EXPECT().Load(gomock.Any(), "room-1").Return(roomAgg, nil).Times(1)
	cache.EXPECT().Get(gomock.Any(), "room:video_call:room-1").Return(mustJSON(t, session), nil).Times(1)

	_, err := service.RelaySignal(context.Background(), apptypes.RelayVideoCallSignalCommand{
		RoomID:          "room-1",
		SessionID:       "session-1",
		ActorID:         "actor-1",
		TargetAccountID: "other-user",
		SignalType:      "offer",
	})
	if !errors.Is(err, entity.ErrVideoCallParticipantNotFound) {
		t.Fatalf("RelaySignal() error = %v, want %v", err, entity.ErrVideoCallParticipantNotFound)
	}
}

func testRoomAggregate(t *testing.T, roomID string, memberIDs ...string) *aggregate.RoomAggregate {
	t.Helper()

	room, err := entity.NewRoom(roomID, "Room", "", memberIDs[0], "group", "", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewRoom() error = %v", err)
	}

	members := make([]*entity.RoomMemberEntity, 0, len(memberIDs))
	for idx, memberID := range memberIDs {
		role := "member"
		if idx == 0 {
			role = "owner"
		}
		member, err := entity.NewRoomMember(
			"member-"+memberID,
			roomID,
			memberID,
			roomtypes.RoomRole(role),
			time.Now().UTC(),
		)
		if err != nil {
			t.Fatalf("NewRoomMember() error = %v", err)
		}
		members = append(members, member)
	}

	agg, err := aggregate.RestoreRoomAggregate(room, members, 0)
	if err != nil {
		t.Fatalf("RestoreRoomAggregate() error = %v", err)
	}
	return agg
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func redisNilWrapped() error {
	return fmt.Errorf("wrapped: %w", redis.Nil)
}
