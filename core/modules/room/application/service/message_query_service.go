package service

import (
	"context"
	"strings"
	"time"

	"wechat-clone/core/modules/room/application/projection"
	roomsupport "wechat-clone/core/modules/room/application/support"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/infra/projection/cassandra/views"
	"wechat-clone/core/shared/pkg/stackErr"

	"golang.org/x/sync/errgroup"
)

const listMessageBuildConcurrency = 16

type MessageQueryService interface {
	ListMessages(ctx context.Context, accountID string, query apptypes.ListMessagesQuery) ([]apptypes.MessageResult, error)
}

type messageQueryService struct {
	readRepos projection.QueryRepos
}

func newMessageQueryService(readRepos projection.QueryRepos) MessageQueryService {
	return &messageQueryService{readRepos: readRepos}
}

func (s *messageQueryService) ListMessages(ctx context.Context, accountID string, query apptypes.ListMessagesQuery) ([]apptypes.MessageResult, error) {
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var beforeAt *time.Time
	if strings.TrimSpace(query.BeforeAt) != "" {
		if parsed, err := time.Parse(time.RFC3339, query.BeforeAt); err == nil {
			beforeAt = &parsed
		}
	}

	messages, err := s.readRepos.MessageReadRepository().ListMessages(ctx, accountID, query.RoomID, projection.MessageListOptions{
		Limit:     limit,
		BeforeID:  query.BeforeID,
		BeforeAt:  beforeAt,
		Ascending: query.Ascending,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	out, err := s.buildMessageResultsConcurrently(ctx, accountID, messages)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return out, nil
}

func (s *messageQueryService) buildMessageResultsConcurrently(
	ctx context.Context,
	accountID string,
	messages []*views.MessageView,
) ([]apptypes.MessageResult, error) {
	results := make([]*apptypes.MessageResult, len(messages))

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(listMessageBuildConcurrency)
	for index, message := range messages {
		index, message := index, message
		if message == nil {
			continue
		}

		eg.Go(func() error {
			item, err := roomsupport.BuildMessageResult(egCtx, s.readRepos, accountID, message)
			if err != nil {
				return stackErr.Error(err)
			}
			if item != nil {
				copyItem := *item
				results[index] = &copyItem
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, stackErr.Error(err)
	}

	out := make([]apptypes.MessageResult, 0, len(results))
	for _, item := range results {
		if item == nil {
			continue
		}
		out = append(out, *item)
	}
	return out, nil
}
