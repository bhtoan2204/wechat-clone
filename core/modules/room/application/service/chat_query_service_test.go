package service

import (
	"context"
	"testing"
	"time"

	"go-socket/core/modules/room/application/projection"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/utils"
)

func TestChatQueryServiceListConversationsSkipsRoomsWithoutViewerMembership(t *testing.T) {
	t.Parallel()

	viewerID := "viewer-account"
	now := time.Date(2026, time.April, 14, 1, 30, 0, 0, time.UTC)

	service := NewChatQueryService(&stubQueryRepos{
		roomRepo: &stubRoomReadRepository{
			listRoomsByAccount: func(_ context.Context, accountID string, _ utils.QueryOptions) ([]*views.RoomView, error) {
				if accountID != viewerID {
					t.Fatalf("unexpected accountID: %s", accountID)
				}
				return []*views.RoomView{
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
				}, nil
			},
		},
		messageRepo: &stubMessageReadRepository{
			countUnreadMessages: func(context.Context, string, string, *time.Time) (int64, error) {
				return 0, nil
			},
			getLastMessage: func(context.Context, string) (*views.MessageView, error) {
				return nil, nil
			},
		},
		memberRepo: &stubRoomMemberReadRepository{
			listRoomMembers: func(_ context.Context, roomID string) ([]*views.RoomMemberView, error) {
				switch roomID {
				case "stale-room":
					return []*views.RoomMemberView{
						{
							ID:        "member-1",
							RoomID:    roomID,
							AccountID: "someone-else",
							Role:      "member",
							CreatedAt: now,
							UpdatedAt: now,
						},
					}, nil
				case "valid-room":
					return []*views.RoomMemberView{
						{
							ID:          "member-2",
							RoomID:      roomID,
							AccountID:   viewerID,
							Role:        "owner",
							DisplayName: "Viewer",
							CreatedAt:   now,
							UpdatedAt:   now,
						},
						{
							ID:          "member-3",
							RoomID:      roomID,
							AccountID:   "peer-account",
							Role:        "member",
							DisplayName: "Peer User",
							CreatedAt:   now,
							UpdatedAt:   now,
						},
					}, nil
				default:
					t.Fatalf("unexpected roomID: %s", roomID)
					return nil, nil
				}
			},
		},
	}, nil)

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

type stubQueryRepos struct {
	roomRepo    projection.RoomReadRepository
	messageRepo projection.MessageReadRepository
	memberRepo  projection.RoomMemberReadRepository
}

func (s *stubQueryRepos) RoomReadRepository() projection.RoomReadRepository {
	return s.roomRepo
}

func (s *stubQueryRepos) MessageReadRepository() projection.MessageReadRepository {
	return s.messageRepo
}

func (s *stubQueryRepos) RoomMemberReadRepository() projection.RoomMemberReadRepository {
	return s.memberRepo
}

type stubRoomReadRepository struct {
	listRoomsByAccount func(ctx context.Context, accountID string, options utils.QueryOptions) ([]*views.RoomView, error)
}

func (s *stubRoomReadRepository) ListRooms(ctx context.Context, options utils.QueryOptions) ([]*views.RoomView, error) {
	return nil, nil
}

func (s *stubRoomReadRepository) ListRoomsByAccount(ctx context.Context, accountID string, options utils.QueryOptions) ([]*views.RoomView, error) {
	if s.listRoomsByAccount == nil {
		return nil, nil
	}
	return s.listRoomsByAccount(ctx, accountID, options)
}

func (s *stubRoomReadRepository) GetRoomByID(ctx context.Context, id string) (*views.RoomView, error) {
	return nil, nil
}

type stubMessageReadRepository struct {
	getMessageByID               func(ctx context.Context, id string) (*views.MessageView, error)
	getLastMessage               func(ctx context.Context, roomID string) (*views.MessageView, error)
	listMessages                 func(ctx context.Context, accountID, roomID string, options projection.MessageListOptions) ([]*views.MessageView, error)
	getMessageReceipt            func(ctx context.Context, messageID, accountID string) (string, *time.Time, *time.Time, error)
	countMessageReceiptsByStatus func(ctx context.Context, messageID, status string) (int64, error)
	countUnreadMessages          func(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error)
}

func (s *stubMessageReadRepository) GetMessageByID(ctx context.Context, id string) (*views.MessageView, error) {
	if s.getMessageByID == nil {
		return nil, nil
	}
	return s.getMessageByID(ctx, id)
}

func (s *stubMessageReadRepository) GetLastMessage(ctx context.Context, roomID string) (*views.MessageView, error) {
	if s.getLastMessage == nil {
		return nil, nil
	}
	return s.getLastMessage(ctx, roomID)
}

func (s *stubMessageReadRepository) ListMessages(
	ctx context.Context,
	accountID,
	roomID string,
	options projection.MessageListOptions,
) ([]*views.MessageView, error) {
	if s.listMessages == nil {
		return nil, nil
	}
	return s.listMessages(ctx, accountID, roomID, options)
}

func (s *stubMessageReadRepository) GetMessageReceipt(ctx context.Context, messageID, accountID string) (string, *time.Time, *time.Time, error) {
	if s.getMessageReceipt == nil {
		return "", nil, nil, nil
	}
	return s.getMessageReceipt(ctx, messageID, accountID)
}

func (s *stubMessageReadRepository) CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error) {
	if s.countMessageReceiptsByStatus == nil {
		return 0, nil
	}
	return s.countMessageReceiptsByStatus(ctx, messageID, status)
}

func (s *stubMessageReadRepository) CountUnreadMessages(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error) {
	if s.countUnreadMessages == nil {
		return 0, nil
	}
	return s.countUnreadMessages(ctx, roomID, accountID, lastReadAt)
}

type stubRoomMemberReadRepository struct {
	listRoomMembers         func(ctx context.Context, roomID string) ([]*views.RoomMemberView, error)
	getRoomMemberByAccount  func(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error)
	searchMentionCandidates func(ctx context.Context, roomID, keyword, excludeAccountID string, limit int) ([]*views.MentionCandidateView, error)
}

func (s *stubRoomMemberReadRepository) ListRoomMembers(ctx context.Context, roomID string) ([]*views.RoomMemberView, error) {
	if s.listRoomMembers == nil {
		return nil, nil
	}
	return s.listRoomMembers(ctx, roomID)
}

func (s *stubRoomMemberReadRepository) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error) {
	if s.getRoomMemberByAccount == nil {
		return nil, nil
	}
	return s.getRoomMemberByAccount(ctx, roomID, accountID)
}

func (s *stubRoomMemberReadRepository) SearchMentionCandidates(
	ctx context.Context,
	roomID,
	keyword,
	excludeAccountID string,
	limit int,
) ([]*views.MentionCandidateView, error) {
	if s.searchMentionCandidates == nil {
		return nil, nil
	}
	return s.searchMentionCandidates(ctx, roomID, keyword, excludeAccountID, limit)
}
