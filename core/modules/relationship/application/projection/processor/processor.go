package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	relationshipprojection "wechat-clone/core/modules/relationship/application/projection"
	relationshipaggregate "wechat-clone/core/modules/relationship/domain/aggregate"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/contracts"
	infraMessaging "wechat-clone/core/shared/infra/messaging"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	consumer []infraMessaging.Consumer
	projRepo relationshipprojection.ReadRepository
}

func NewProcessor(cfg *config.Config, projRepo relationshipprojection.ReadRepository) (Processor, error) {
	instance := &processor{
		consumer: make([]infraMessaging.Consumer, 0, 1),
		projRepo: projRepo,
	}

	topic := strings.TrimSpace(cfg.KafkaConfig.KafkaRelationshipConsumer.RelationshipOutboxTopic)
	if topic == "" || projRepo == nil {
		return instance, nil
	}

	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaRelationshipConsumer.RelationshipProjectionGroup,
		OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
		ConsumeTopic: []string{topic},
		HandlerName:  fmt.Sprintf("relationship-projection-%s-handler", strings.ToLower(topic)),
		DLQ:          true,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	consumer.SetHandler(func(ctx context.Context, value []byte) error {
		return instance.handleRelationshipOutboxEvent(ctx, value)
	})
	instance.consumer = append(instance.consumer, consumer)

	return instance, nil
}

func (p *processor) Start() error {
	for _, consumer := range p.consumer {
		consumer.Read(infraMessaging.WrapConsumerCallback(consumer, "Handle relationship projection message failed"))
	}
	return nil
}

func (p *processor) Stop() error {
	infraMessaging.StopConsumers(p.consumer)
	return nil
}

func (p *processor) handleRelationshipOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("RelationshipProjection")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal relationship outbox event failed: %w", err))
	}

	log.Infow("handle relationship outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case relationshipprojection.EventRelationshipPairFriendRequestSent:
		return p.projectFriendRequestSent(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairFriendRequestCancelled:
		return p.projectFriendRequestCancelled(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairFriendRequestRejected:
		return p.projectFriendRequestRejected(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairFriendRequestAccepted:
		return p.projectFriendRequestAccepted(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairFollowed:
		return p.projectFollowed(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairUnfollowed:
		return p.projectUnfollowed(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairBlocked:
		return p.projectBlocked(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairUnblocked:
		return p.projectUnblocked(ctx, event.EventData)
	case relationshipprojection.EventRelationshipPairUnfriended:
		return p.projectUnfriended(ctx, event.EventData)
	default:
		return nil
	}
}

func (p *processor) projectFriendRequestSent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairFriendRequestSent, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship friend request sent payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairFriendRequestSent)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairFriendRequestSent))
	}
	projection, err := p.loadProjection(ctx, payload.RequesterID, payload.AddresseeID, payload.CreatedAt)
	if err != nil {
		return stackErr.Error(err)
	}
	projection.PendingRequestID = payload.RequestID
	projection.PendingRequesterID = payload.RequesterID
	projection.PendingAddresseeID = payload.AddresseeID
	projection.PendingRequestCreatedAt = timePtr(payload.CreatedAt)
	projection.UpdatedAt = payload.CreatedAt.UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectFriendRequestAccepted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairFriendRequestAccepted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship friend request accepted payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairFriendRequestAccepted)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairFriendRequestAccepted))
	}
	projection, err := p.loadProjection(ctx, payload.RequesterID, payload.AddresseeID, payload.CreatedAt)
	if err != nil {
		return stackErr.Error(err)
	}
	clearPendingRequest(projection)
	projection.FriendshipID = payload.FriendshipID
	projection.FriendshipCreatedAt = timePtr(payload.AcceptedAt)
	projection.UpdatedAt = payload.AcceptedAt.UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectFollowed(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairFollowed, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship followed payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairFollowed)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairFollowed))
	}
	projection, err := p.loadProjection(ctx, payload.FollowerID, payload.FolloweeID, payload.CreatedAt)
	if err != nil {
		return stackErr.Error(err)
	}
	setDirectionalBool(projection, payload.FollowerID, payload.FolloweeID, payload.CreatedAt.UTC(), directionFieldFollow)
	projection.UpdatedAt = payload.CreatedAt.UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectUnfollowed(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairUnfollowed, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship unfollowed payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairUnfollowed)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairUnfollowed))
	}
	projection, err := p.loadProjection(ctx, payload.FollowerID, payload.FolloweeID, time.Now().UTC())
	if err != nil {
		return stackErr.Error(err)
	}
	clearDirectionalBool(projection, payload.FollowerID, payload.FolloweeID, directionFieldFollow)
	projection.UpdatedAt = time.Now().UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectBlocked(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairBlocked, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship blocked payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairBlocked)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairBlocked))
	}
	projection, err := p.loadProjection(ctx, payload.BlockerID, payload.BlockedID, payload.CreatedAt)
	if err != nil {
		return stackErr.Error(err)
	}
	clearPendingRequest(projection)
	setDirectionalBool(projection, payload.BlockerID, payload.BlockedID, payload.CreatedAt.UTC(), directionFieldBlock)
	projection.UpdatedAt = payload.CreatedAt.UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectUnblocked(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairUnblocked, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship unblocked payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairUnblocked)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairUnblocked))
	}
	projection, err := p.loadProjection(ctx, payload.BlockerID, payload.BlockedID, time.Now().UTC())
	if err != nil {
		return stackErr.Error(err)
	}
	clearDirectionalBool(projection, payload.BlockerID, payload.BlockedID, directionFieldBlock)
	projection.UpdatedAt = time.Now().UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectUnfriended(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairUnfriended, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode relationship unfriended payload failed: %w", err))
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairUnfriended)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairUnfriended))
	}
	projection, err := p.loadProjection(ctx, payload.UserID, payload.FriendID, time.Now().UTC())
	if err != nil {
		return stackErr.Error(err)
	}
	projection.FriendshipID = ""
	projection.FriendshipCreatedAt = nil
	projection.UpdatedAt = time.Now().UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectFriendRequestCancelled(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairFriendRequestCancelled, raw)
	if err != nil {
		return stackErr.Error(err)
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairFriendRequestCancelled)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairFriendRequestCancelled))
	}
	projection, err := p.loadProjection(ctx, payload.RequesterID, payload.AddresseeID, payload.CancelledAt)
	if err != nil {
		return stackErr.Error(err)
	}
	if projection.PendingRequestID == payload.RequestID {
		clearPendingRequest(projection)
	}
	projection.UpdatedAt = payload.CancelledAt.UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) projectFriendRequestRejected(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, relationshipprojection.EventRelationshipPairFriendRequestRejected, raw)
	if err != nil {
		return stackErr.Error(err)
	}
	payload, ok := payloadAny.(*relationshipaggregate.EventRelationshipPairFriendRequestRejected)
	if !ok || payload == nil {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", relationshipprojection.EventRelationshipPairFriendRequestRejected))
	}
	projection, err := p.loadProjection(ctx, payload.RequesterID, payload.AddresseeID, payload.RejectedAt)
	if err != nil {
		return stackErr.Error(err)
	}
	if projection.PendingRequestID == payload.RequestID {
		clearPendingRequest(projection)
	}
	projection.UpdatedAt = payload.RejectedAt.UTC()
	return stackErr.Error(p.projRepo.SavePair(ctx, projection))
}

func (p *processor) loadProjection(ctx context.Context, userA, userB string, now time.Time) (*relationshipprojection.RelationshipPairProjection, error) {
	projection, err := p.projRepo.GetPair(ctx, userA, userB)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if projection != nil {
		return projection, nil
	}
	low, high := normalizePair(strings.TrimSpace(userA), strings.TrimSpace(userB))
	return &relationshipprojection.RelationshipPairProjection{
		PairID:     low + ":" + high,
		UserLowID:  low,
		UserHighID: high,
		CreatedAt:  now.UTC(),
		UpdatedAt:  now.UTC(),
	}, nil
}

type directionField int

const (
	directionFieldFollow directionField = iota
	directionFieldBlock
)

func setDirectionalBool(projection *relationshipprojection.RelationshipPairProjection, actorID, targetID string, at time.Time, field directionField) {
	low, high := normalizePair(actorID, targetID)
	isLowToHigh := actorID == low && targetID == high
	switch field {
	case directionFieldFollow:
		if isLowToHigh {
			projection.LowFollowsHigh = true
			projection.LowFollowsHighAt = timePtr(at)
			return
		}
		projection.HighFollowsLow = true
		projection.HighFollowsLowAt = timePtr(at)
	case directionFieldBlock:
		if isLowToHigh {
			projection.LowBlocksHigh = true
			projection.LowBlocksHighAt = timePtr(at)
			return
		}
		projection.HighBlocksLow = true
		projection.HighBlocksLowAt = timePtr(at)
	}
}

func clearDirectionalBool(projection *relationshipprojection.RelationshipPairProjection, actorID, targetID string, field directionField) {
	low, high := normalizePair(actorID, targetID)
	isLowToHigh := actorID == low && targetID == high
	switch field {
	case directionFieldFollow:
		if isLowToHigh {
			projection.LowFollowsHigh = false
			projection.LowFollowsHighAt = nil
			return
		}
		projection.HighFollowsLow = false
		projection.HighFollowsLowAt = nil
	case directionFieldBlock:
		if isLowToHigh {
			projection.LowBlocksHigh = false
			projection.LowBlocksHighAt = nil
			return
		}
		projection.HighBlocksLow = false
		projection.HighBlocksLowAt = nil
	}
}

func clearPendingRequest(projection *relationshipprojection.RelationshipPairProjection) {
	projection.PendingRequestID = ""
	projection.PendingRequesterID = ""
	projection.PendingAddresseeID = ""
	projection.PendingRequestCreatedAt = nil
}

func timePtr(value time.Time) *time.Time {
	utc := value.UTC()
	return &utc
}

func normalizePair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
