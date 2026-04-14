package service

import (
	"context"
	"testing"
	"time"

	"go-socket/core/modules/room/application/projection"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/utils"

	"go.uber.org/mock/gomock"
)

func TestChatQueryServiceListConversationsSkipsRoomsWithoutViewerMembership(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	viewerID := "viewer-account"
	now := time.Date(2026, time.April, 14, 1, 30, 0, 0, time.UTC)

	queryRepos := projection.NewMockQueryRepos(ctrl)
	roomRepo := projection.NewMockRoomReadRepository(ctrl)
	messageRepo := projection.NewMockMessageReadRepository(ctrl)
	memberRepo := projection.NewMockRoomMemberReadRepository(ctrl)

	queryRepos.EXPECT().RoomReadRepository().Return(roomRepo).AnyTimes()
	queryRepos.EXPECT().MessageReadRepository().Return(messageRepo).AnyTimes()
	queryRepos.EXPECT().RoomMemberReadRepository().Return(memberRepo).AnyTimes()

	roomRepo.EXPECT().
		ListRoomsByAccount(gomock.Any(), viewerID, gomock.AssignableToTypeOf(utils.QueryOptions{})).
		Return([]*views.RoomView{
			{
				ID:        "stale-room",
				Name:      "Stale room",
				RoomType:  "direct",
				OwnerID:   viewerID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "valid-room",
				Name:      "Valid room",
				RoomType:  "direct",
				OwnerID:   viewerID,
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil).
		Times(1)

	messageRepo.EXPECT().
		CountUnreadMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(int64(0), nil).
		AnyTimes()

	messageRepo.EXPECT().
		GetLastMessage(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		AnyTimes()

	memberRepo.EXPECT().
		ListRoomMembers(gomock.Any(), "stale-room").
		Return([]*views.RoomMemberView{
			{
				ID:        "member-1",
				RoomID:    "stale-room",
				AccountID: "someone-else",
				Role:      "member",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil).
		Times(1)

	memberRepo.EXPECT().
		ListRoomMembers(gomock.Any(), "valid-room").
		Return([]*views.RoomMemberView{
			{
				ID:          "member-2",
				RoomID:      "valid-room",
				AccountID:   viewerID,
				Role:        "owner",
				DisplayName: "Viewer",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "member-3",
				RoomID:      "valid-room",
				AccountID:   "peer-account",
				Role:        "member",
				DisplayName: "Peer User",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}, nil).
		Times(1)

	service := NewChatQueryService(queryRepos, nil)

	results, err := service.ListConversations(context.Background(), viewerID, apptypes.ListConversationsQuery{
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListConversations() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 conversation result, got %d", len(results))
	}
	if results[0].RoomID != "valid-room" {
		t.Fatalf("expected valid room to remain, got %s", results[0].RoomID)
	}
	if results[0].Name != "Peer User" {
		t.Fatalf("expected direct room name to resolve from peer member, got %s", results[0].Name)
	}
}
