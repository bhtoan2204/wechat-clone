package service

import (
	"context"
	"errors"
	"strings"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
)

func (s *RoomCommandService) CreateRoom(ctx context.Context, accountID string, command apptypes.CreateRoomCommand) (*apptypes.RoomResult, error) {
	accountID = strings.TrimSpace(accountID)
	name := strings.TrimSpace(command.Name)
	if accountID == "" || name == "" {
		return nil, errors.New("account_id and name are required")
	}

	now := time.Now().UTC()
	room := &entity.Room{
		ID:          newUUID(),
		Name:        name,
		Description: strings.TrimSpace(command.Description),
		RoomType:    command.RoomType,
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
		return s.aggregateService.PublishRoomCreated(ctx, txRepos.RoomOutboxEventsRepository(), room.ID, room.RoomType, 1)
	}); err != nil {
		return nil, err
	}

	return buildRoomResult(room), nil
}

func (s *RoomCommandService) UpdateRoom(ctx context.Context, accountID, roomID string, command apptypes.UpdateRoomCommand) (*apptypes.RoomResult, error) {
	room, err := s.repos.RoomRepository().GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(command.Name); name != "" {
		room.Name = name
	}
	if description := strings.TrimSpace(command.Description); description != "" {
		room.Description = description
	}
	if command.RoomType != "" {
		room.RoomType = command.RoomType
	}
	if accountID = strings.TrimSpace(accountID); accountID != "" {
		room.OwnerID = accountID
	}
	room.UpdatedAt = time.Now().UTC()
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
			return err
		}
		return txRepos.RoomReadRepository().UpdateRoom(ctx, room)
	}); err != nil {
		return nil, err
	}

	return buildRoomResult(room), nil
}

func (s *RoomCommandService) DeleteRoom(ctx context.Context, roomID string) error {
	return s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().DeleteRoom(ctx, roomID); err != nil {
			return err
		}
		return txRepos.RoomReadRepository().DeleteRoom(ctx, roomID)
	})
}

func (s *RoomCommandService) JoinRoom(ctx context.Context, accountID string, command apptypes.JoinRoomCommand) error {
	return errors.New("not implemented")
}
