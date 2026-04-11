package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	roomtypes "go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/stackErr"
)

func (s *RoomCommandService) CreateDirectConversation(ctx context.Context, accountID string, command apptypes.CreateDirectConversationCommand) (*apptypes.ConversationResult, error) {
	peerID := strings.TrimSpace(command.PeerAccountID)
	now := time.Now().UTC()
	room, err := entity.NewDirectConversationRoom(newUUID(), accountID, peerID, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if existing, err := s.repos.RoomRepository().GetRoomByDirectKey(ctx, room.DirectKey); err == nil && existing != nil {
		return buildConversationResult(ctx, s.repos, accountID, existing, true)
	}
	ownerMember, err := entity.NewRoomMember(newUUID(), room.ID, accountID, roomtypes.RoomRoleOwner, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	peerMember, err := entity.NewRoomMember(newUUID(), room.ID, peerID, roomtypes.RoomRoleMember, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	members := []*entity.RoomMemberEntity{ownerMember, peerMember}

	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().CreateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomReadRepository().UpsertRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		for _, member := range members {
			if err := txRepos.RoomMemberRepository().CreateRoomMember(ctx, member); err != nil {
				return stackErr.Error(err)
			}
			if err := txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, member); err != nil {
				return stackErr.Error(err)
			}
			if err := s.aggregateService.PublishMemberAdded(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, member.AccountID, member.Role, member.CreatedAt); err != nil {
				return stackErr.Error(err)
			}
		}
		if err := s.aggregateService.PublishRoomCreated(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, room.RoomType, len(members)); err != nil {
			return stackErr.Error(err)
		}
		message, err := createSystemMessageTx(ctx, txRepos, room.ID, accountID, fmt.Sprintf("%s started a direct conversation", accountID), now)
		if err != nil {
			return stackErr.Error(err)
		}
		return stackErr.Error(txRepos.RoomReadRepository().UpdateRoomStats(ctx, room.ID, len(members), message, now))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *RoomCommandService) CreateGroup(ctx context.Context, accountID string, command apptypes.CreateGroupCommand) (*apptypes.ConversationResult, error) {
	now := time.Now().UTC()
	room, err := entity.NewRoom(newUUID(), command.Name, command.Description, accountID, roomtypes.RoomTypeGroup, "", now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	memberSet, err := entity.BuildGroupMemberRoles(accountID, command.MemberIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().CreateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomReadRepository().UpsertRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}

		memberCount := 0
		for memberID, role := range memberSet {
			member, err := entity.NewRoomMember(newUUID(), room.ID, memberID, role, now)
			if err != nil {
				return stackErr.Error(err)
			}
			if err := txRepos.RoomMemberRepository().CreateRoomMember(ctx, member); err != nil {
				return stackErr.Error(err)
			}
			if err := txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, member); err != nil {
				return stackErr.Error(err)
			}
			if err := s.aggregateService.PublishMemberAdded(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, member.AccountID, member.Role, member.CreatedAt); err != nil {
				return stackErr.Error(err)
			}
			memberCount++
		}

		if err := s.aggregateService.PublishRoomCreated(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, room.RoomType, memberCount); err != nil {
			return stackErr.Error(err)
		}
		message, err := createSystemMessageTx(ctx, txRepos, room.ID, accountID, fmt.Sprintf("%s created the group", accountID), now)
		if err != nil {
			return stackErr.Error(err)
		}
		return stackErr.Error(txRepos.RoomReadRepository().UpdateRoomStats(ctx, room.ID, memberCount, message, now))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *RoomCommandService) UpdateGroup(ctx context.Context, accountID, roomID string, command apptypes.UpdateGroupCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := member.CanManageGroup(room); err != nil {
		return nil, stackErr.Error(err)
	}

	updated, err := room.UpdateDetails(command.Name, command.Description, "", time.Now().UTC())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if !updated {
		return buildConversationResult(ctx, s.repos, accountID, room, true)
	}

	now := time.Now().UTC()
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		room.Touch(now)
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomReadRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}

		message, err := createSystemMessageTx(ctx, txRepos, room.ID, accountID, fmt.Sprintf("group renamed to %s", room.Name), now)
		if err != nil {
			return stackErr.Error(err)
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, room.ID)
		if err != nil {
			return stackErr.Error(err)
		}
		return stackErr.Error(txRepos.RoomReadRepository().UpdateRoomStats(ctx, room.ID, len(members), message, now))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *RoomCommandService) AddMember(ctx context.Context, actorID, roomID string, command apptypes.AddMemberCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, actorID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := member.CanManageGroup(room); err != nil {
		return nil, stackErr.Error(err)
	}

	accountID := strings.TrimSpace(command.AccountID)
	if accountID == "" {
		return nil, stackErr.Error(errors.New("account_id is required"))
	}

	if existing, err := s.repos.RoomMemberRepository().GetRoomMemberByAccount(ctx, roomID, accountID); err == nil && existing != nil {
		return buildConversationResult(ctx, s.repos, actorID, room, true)
	}

	now := time.Now().UTC()
	newMember, err := entity.NewRoomMember(newUUID(), roomID, accountID, command.Role, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomMemberRepository().CreateRoomMember(ctx, newMember); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, newMember); err != nil {
			return stackErr.Error(err)
		}
		if err := s.aggregateService.PublishMemberAdded(ctx, txRepos.RoomOutboxEventsRepository(), roomID, newMember.AccountID, newMember.Role, newMember.CreatedAt); err != nil {
			return stackErr.Error(err)
		}

		room.Touch(now)
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomReadRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}

		message, err := createSystemMessageTx(ctx, txRepos, roomID, actorID, fmt.Sprintf("%s joined", accountID), now)
		if err != nil {
			return stackErr.Error(err)
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
		if err != nil {
			return stackErr.Error(err)
		}
		return stackErr.Error(txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildConversationResult(ctx, s.repos, actorID, room, true)
}

func (s *RoomCommandService) RemoveMember(ctx context.Context, actorID, roomID string, command apptypes.RemoveMemberCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, actorID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountID := strings.TrimSpace(command.AccountID)
	if err := member.CanRemoveFrom(room, accountID); err != nil {
		return nil, stackErr.Error(err)
	}

	removedMember, err := s.repos.RoomMemberRepository().GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomMemberRepository().DeleteRoomMember(ctx, roomID, accountID); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomMemberReadRepository().DeleteRoomMember(ctx, roomID, accountID); err != nil {
			return stackErr.Error(err)
		}
		if err := s.aggregateService.PublishMemberRemoved(ctx, txRepos.RoomOutboxEventsRepository(), roomID, removedMember.AccountID, removedMember.Role, now); err != nil {
			return stackErr.Error(err)
		}

		room.Touch(now)
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomReadRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}

		message, err := createSystemMessageTx(ctx, txRepos, roomID, actorID, fmt.Sprintf("%s left", accountID), now)
		if err != nil {
			return stackErr.Error(err)
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
		if err != nil {
			return stackErr.Error(err)
		}
		return stackErr.Error(txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildConversationResult(ctx, s.repos, actorID, room, true)
}

func (s *RoomCommandService) PinMessage(ctx context.Context, actorID, roomID string, command apptypes.PinMessageCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, actorID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := member.CanManageGroup(room); err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	if err := room.PinMessage(command.MessageID, now); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RoomReadRepository().UpdatePinnedMessage(ctx, roomID, room.PinnedMessageID, now); err != nil {
			return stackErr.Error(err)
		}

		message, err := createSystemMessageTx(ctx, txRepos, roomID, actorID, fmt.Sprintf("message %s pinned", room.PinnedMessageID), now)
		if err != nil {
			return stackErr.Error(err)
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
		if err != nil {
			return stackErr.Error(err)
		}
		return stackErr.Error(txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildConversationResult(ctx, s.repos, actorID, room, true)
}
