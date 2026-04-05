package service

import "go-socket/core/modules/room/domain/repos"

type MessageCommandService struct {
	repos            repos.Repos
	aggregateService *RoomAggregateService
}

func NewMessageCommandService(repos repos.Repos, aggregateService *RoomAggregateService) *MessageCommandService {
	return &MessageCommandService{
		repos:            repos,
		aggregateService: aggregateService,
	}
}
