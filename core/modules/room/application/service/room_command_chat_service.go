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
)

func (s *RoomCommandService) CreateDirectConversation(ctx context.Context, accountID string, command apptypes.CreateDirectConversationCommand) (*apptypes.ConversationResult, error) {
	accountID = strings.TrimSpace(accountID)
	peerID := strings.TrimSpace(command.PeerAccountID)
	if accountID == "" || peerID == "" {
		return nil, errors.New("account_id and peer_account_id are required")
	}
	if accountID == peerID {
		return nil, errors.New("cannot create direct conversation with yourself")
	}

	directKey := canonicalDirectKey(accountID, peerID)
	if existing, err := s.repos.RoomRepository().GetRoomByDirectKey(ctx, directKey); err == nil && existing != nil {
		return buildConversationResult(ctx, s.repos, accountID, existing, true)
	}

	now := time.Now().UTC()
	room := &entity.Room{
		ID:          newUUID(),
		Name:        "Direct chat",
		Description: "",
		RoomType:    roomtypes.RoomTypeDirect,
		OwnerID:     accountID,
		DirectKey:   directKey,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	members := []*entity.RoomMemberEntity{
		{ID: newUUID(), RoomID: room.ID, AccountID: accountID, Role: roomtypes.RoomRoleOwner, CreatedAt: now, UpdatedAt: now},
		{ID: newUUID(), RoomID: room.ID, AccountID: peerID, Role: roomtypes.RoomRoleMember, CreatedAt: now, UpdatedAt: now},
	}

	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().CreateRoom(ctx, room); err != nil {
			return err
		}
		if err := txRepos.RoomReadRepository().UpsertRoom(ctx, room); err != nil {
			return err
		}
		for _, member := range members {
			if err := txRepos.RoomMemberRepository().CreateRoomMember(ctx, member); err != nil {
				return err
			}
			if err := txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, member); err != nil {
				return err
			}
			if err := s.aggregateService.PublishMemberAdded(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, member.AccountID, member.Role, member.CreatedAt); err != nil {
				return err
			}
		}
		if err := s.aggregateService.PublishRoomCreated(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, room.RoomType, len(members)); err != nil {
			return err
		}
		message, err := createSystemMessageTx(ctx, txRepos, room.ID, accountID, fmt.Sprintf("%s started a direct conversation", accountID), now)
		if err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoomStats(ctx, room.ID, len(members), message, now)
	}); err != nil {
		return nil, err
	}

	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *RoomCommandService) CreateGroup(ctx context.Context, accountID string, command apptypes.CreateGroupCommand) (*apptypes.ConversationResult, error) {
	accountID = strings.TrimSpace(accountID)
	name := strings.TrimSpace(command.Name)
	if accountID == "" || name == "" {
		return nil, errors.New("account_id and name are required")
	}

	memberSet := map[string]roomtypes.RoomRole{accountID: roomtypes.RoomRoleOwner}
	for _, memberID := range command.MemberIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			continue
		}
		if _, exists := memberSet[memberID]; !exists {
			memberSet[memberID] = roomtypes.RoomRoleMember
		}
	}

	now := time.Now().UTC()
	room := &entity.Room{
		ID:          newUUID(),
		Name:        name,
		Description: strings.TrimSpace(command.Description),
		RoomType:    roomtypes.RoomTypeGroup,
		OwnerID:     accountID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().CreateRoom(ctx, room); err != nil {
			return err
		}
		if err := txRepos.RoomReadRepository().UpsertRoom(ctx, room); err != nil {
			return err
		}

		memberCount := 0
		for memberID, role := range memberSet {
			member := &entity.RoomMemberEntity{
				ID:        newUUID(),
				RoomID:    room.ID,
				AccountID: memberID,
				Role:      role,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := txRepos.RoomMemberRepository().CreateRoomMember(ctx, member); err != nil {
				return err
			}
			if err := txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, member); err != nil {
				return err
			}
			if err := s.aggregateService.PublishMemberAdded(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, member.AccountID, member.Role, member.CreatedAt); err != nil {
				return err
			}
			memberCount++
		}

		if err := s.aggregateService.PublishRoomCreated(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, room.RoomType, memberCount); err != nil {
			return err
		}
		message, err := createSystemMessageTx(ctx, txRepos, room.ID, accountID, fmt.Sprintf("%s created the group", accountID), now)
		if err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoomStats(ctx, room.ID, memberCount, message, now)
	}); err != nil {
		return nil, err
	}

	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *RoomCommandService) UpdateGroup(ctx context.Context, accountID, roomID string, command apptypes.UpdateGroupCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, accountID)
	if err != nil {
		return nil, err
	}
	if room.RoomType != roomtypes.RoomTypeGroup {
		return nil, errors.New("room is not a group")
	}
	if member.Role != roomtypes.RoomRoleOwner && member.Role != roomtypes.RoomRoleAdmin {
		return nil, errors.New("insufficient permissions")
	}

	updated := false
	if name := strings.TrimSpace(command.Name); name != "" && name != room.Name {
		room.Name = name
		updated = true
	}
	if description := strings.TrimSpace(command.Description); description != room.Description {
		room.Description = description
		updated = true
	}
	if !updated {
		return buildConversationResult(ctx, s.repos, accountID, room, true)
	}

	now := time.Now().UTC()
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		room.UpdatedAt = now
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}
		if err := txRepos.RoomReadRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}

		message, err := createSystemMessageTx(ctx, txRepos, room.ID, accountID, fmt.Sprintf("group renamed to %s", room.Name), now)
		if err != nil {
			return err
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, room.ID)
		if err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoomStats(ctx, room.ID, len(members), message, now)
	}); err != nil {
		return nil, err
	}

	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *RoomCommandService) AddMember(ctx context.Context, actorID, roomID string, command apptypes.AddMemberCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, actorID)
	if err != nil {
		return nil, err
	}
	if room.RoomType != roomtypes.RoomTypeGroup {
		return nil, errors.New("room is not a group")
	}
	if member.Role != roomtypes.RoomRoleOwner && member.Role != roomtypes.RoomRoleAdmin {
		return nil, errors.New("insufficient permissions")
	}

	accountID := strings.TrimSpace(command.AccountID)
	if accountID == "" {
		return nil, errors.New("account_id is required")
	}
	if command.Role == "" {
		command.Role = roomtypes.RoomRoleMember
	}

	if existing, err := s.repos.RoomMemberRepository().GetRoomMemberByAccount(ctx, roomID, accountID); err == nil && existing != nil {
		return buildConversationResult(ctx, s.repos, actorID, room, true)
	}

	now := time.Now().UTC()
	newMember := &entity.RoomMemberEntity{
		ID:        newUUID(),
		RoomID:    roomID,
		AccountID: accountID,
		Role:      command.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomMemberRepository().CreateRoomMember(ctx, newMember); err != nil {
			return err
		}
		if err := txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, newMember); err != nil {
			return err
		}
		if err := s.aggregateService.PublishMemberAdded(ctx, txRepos.RoomOutboxEventsRepository(), roomID, newMember.AccountID, newMember.Role, newMember.CreatedAt); err != nil {
			return err
		}

		room.UpdatedAt = now
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}
		if err := txRepos.RoomReadRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}

		message, err := createSystemMessageTx(ctx, txRepos, roomID, actorID, fmt.Sprintf("%s joined", accountID), now)
		if err != nil {
			return err
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
		if err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now)
	}); err != nil {
		return nil, err
	}

	return buildConversationResult(ctx, s.repos, actorID, room, true)
}

func (s *RoomCommandService) RemoveMember(ctx context.Context, actorID, roomID string, command apptypes.RemoveMemberCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, actorID)
	if err != nil {
		return nil, err
	}
	if room.RoomType != roomtypes.RoomTypeGroup {
		return nil, errors.New("room is not a group")
	}

	accountID := strings.TrimSpace(command.AccountID)
	if accountID == "" {
		return nil, errors.New("account_id is required")
	}
	if actorID != accountID && member.Role != roomtypes.RoomRoleOwner && member.Role != roomtypes.RoomRoleAdmin {
		return nil, errors.New("insufficient permissions")
	}
	if actorID == accountID && member.Role == roomtypes.RoomRoleOwner {
		return nil, errors.New("owner cannot leave without transferring ownership")
	}

	removedMember, err := s.repos.RoomMemberRepository().GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomMemberRepository().DeleteRoomMember(ctx, roomID, accountID); err != nil {
			return err
		}
		if err := txRepos.RoomMemberReadRepository().DeleteRoomMember(ctx, roomID, accountID); err != nil {
			return err
		}
		if err := s.aggregateService.PublishMemberRemoved(ctx, txRepos.RoomOutboxEventsRepository(), roomID, removedMember.AccountID, removedMember.Role, now); err != nil {
			return err
		}

		room.UpdatedAt = now
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}
		if err := txRepos.RoomReadRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}

		message, err := createSystemMessageTx(ctx, txRepos, roomID, actorID, fmt.Sprintf("%s left", accountID), now)
		if err != nil {
			return err
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
		if err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now)
	}); err != nil {
		return nil, err
	}

	return buildConversationResult(ctx, s.repos, actorID, room, true)
}

func (s *RoomCommandService) PinMessage(ctx context.Context, actorID, roomID string, command apptypes.PinMessageCommand) (*apptypes.ConversationResult, error) {
	member, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, actorID)
	if err != nil {
		return nil, err
	}
	if room.RoomType != roomtypes.RoomTypeGroup {
		return nil, errors.New("room is not a group")
	}
	if member.Role != roomtypes.RoomRoleOwner && member.Role != roomtypes.RoomRoleAdmin {
		return nil, errors.New("insufficient permissions")
	}

	messageID := strings.TrimSpace(command.MessageID)
	if messageID == "" {
		return nil, errors.New("message_id is required")
	}

	now := time.Now().UTC()
	room.PinnedMessageID = messageID
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		room.UpdatedAt = now
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}
		if err := txRepos.RoomReadRepository().UpdatePinnedMessage(ctx, roomID, messageID, now); err != nil {
			return err
		}

		message, err := createSystemMessageTx(ctx, txRepos, roomID, actorID, fmt.Sprintf("message %s pinned", messageID), now)
		if err != nil {
			return err
		}
		members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
		if err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now)
	}); err != nil {
		return nil, err
	}

	return buildConversationResult(ctx, s.repos, actorID, room, true)
}
