package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/payment/domain/entity"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (p *processor) handleAccountEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("PaymentAccountProjection")
	var event accountOutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackerr.Error(fmt.Errorf("unmarshal account outbox event failed: %w", err))
	}

	log.Infow("handle account event", zap.String("event_name", event.EventName))
	switch event.EventName {
	case "EventAccountCreated":
		return p.handleAccountCreatedEvent(ctx, event.EventData)
	case "EventAccountUpdated":
		return p.handleAccountUpdatedEvent(ctx, event.EventData)
	case "EventAccountBanned":
		return p.handleAccountBannedEvent(ctx, event.EventData)
	default:
		return nil
	}
}

func (p *processor) handleAccountCreatedEvent(ctx context.Context, raw json.RawMessage) error {
	log := logging.FromContext(ctx).Named("handleAccountCreatedEvent")
	payloadAny, err := decodeEventPayload(ctx, "EventAccountCreated", raw)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode event payload failed: %w", err))
	}
	payload, ok := payloadAny.(*aggregate.EventAccountCreated)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid payload type for event %s", "EventAccountCreated"))
	}

	existing, err := p.accountProjectionRepo.GetAccountProjectionByAccountID(ctx, payload.AccountID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Errorw("get account projection failed", zap.Error(err))
		return stackerr.Error(fmt.Errorf("get account projection failed: %w", err))
	}

	projection := &entity.PaymentAccount{
		ID:        payload.AccountID,
		AccountID: payload.AccountID,
		Email:     payload.Email,
		CreatedAt: payload.CreatedAt,
		UpdatedAt: payload.CreatedAt,
	}

	if existing == nil {
		if err := p.accountProjectionRepo.CreateAccountProjection(ctx, projection); err != nil {
			log.Errorw("create account projection failed", zap.Error(err))
			return stackerr.Error(fmt.Errorf("create account projection failed: %w", err))
		}
		return nil
	}

	existing.Email = payload.Email
	existing.AccountID = payload.AccountID
	if err := p.accountProjectionRepo.UpdateAccountProjection(ctx, existing); err != nil {
		log.Errorw("update account projection failed", zap.Error(err))
		return stackerr.Error(fmt.Errorf("update account projection failed: %w", err))
	}
	return nil
}

func (p *processor) handleAccountUpdatedEvent(ctx context.Context, raw json.RawMessage) error {
	log := logging.FromContext(ctx).Named("handleAccountUpdatedEvent")
	payloadAny, err := decodeEventPayload(ctx, "EventAccountUpdated", raw)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode event payload failed: %w", err))
	}
	payload, ok := payloadAny.(*aggregate.EventAccountUpdated)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid payload type for event %s", "EventAccountUpdated"))
	}

	existing, err := p.accountProjectionRepo.GetAccountProjectionByAccountID(ctx, payload.AccountID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Errorw("get account projection failed", zap.Error(err))
		return stackerr.Error(fmt.Errorf("get account projection failed: %w", err))
	}

	if existing == nil {
		projection := &entity.PaymentAccount{
			ID:        payload.AccountID,
			AccountID: payload.AccountID,
			Email:     payload.Email,
			CreatedAt: payload.UpdatedAt,
			UpdatedAt: payload.UpdatedAt,
		}
		if err := p.accountProjectionRepo.CreateAccountProjection(ctx, projection); err != nil {
			log.Errorw("create account projection failed", zap.Error(err))
			return stackerr.Error(fmt.Errorf("create account projection failed: %w", err))
		}
		return nil
	}

	existing.Email = payload.Email
	existing.AccountID = payload.AccountID
	existing.UpdatedAt = payload.UpdatedAt
	if err := p.accountProjectionRepo.UpdateAccountProjection(ctx, existing); err != nil {
		log.Errorw("update account projection failed", zap.Error(err))
		return stackerr.Error(fmt.Errorf("update account projection failed: %w", err))
	}
	return nil
}

func (p *processor) handleAccountBannedEvent(ctx context.Context, raw json.RawMessage) error {
	log := logging.FromContext(ctx).Named("handleAccountBannedEvent")
	payloadAny, err := decodeEventPayload(ctx, "EventAccountBanned", raw)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode event payload failed: %w", err))
	}
	payload, ok := payloadAny.(*aggregate.EventAccountBanned)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid payload type for event %s", "EventAccountBanned"))
	}

	if err := p.accountProjectionRepo.DeleteAccountProjection(ctx, payload.AccountID); err != nil {
		log.Errorw("delete account projection failed", zap.Error(err))
		return stackerr.Error(fmt.Errorf("delete account projection failed: %w", err))
	}
	return nil
}
