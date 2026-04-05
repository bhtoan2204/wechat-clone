package service

import (
	"go-socket/core/modules/room/domain/repos"
)

type RoomCommandService struct {
	repos            repos.Repos
	aggregateService *RoomAggregateService
}

func NewRoomCommandService(repos repos.Repos, aggregateService *RoomAggregateService) *RoomCommandService {
	return &RoomCommandService{
		repos:            repos,
		aggregateService: aggregateService,
	}
}
