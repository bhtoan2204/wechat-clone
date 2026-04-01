package projection

import (
	"context"
	"encoding/json"
	"fmt"
	accountaggregate "go-socket/core/modules/account/domain/aggregate"
	paymentaggregate "go-socket/core/modules/payment/domain/aggregate"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"reflect"
	"strings"

	"go.uber.org/zap"
)

func newProjectionSerializer() (eventpkg.Serializer, error) {
	serializer := eventpkg.NewSerializer()
	aggregates := []eventpkg.BaseAggregate{
		&accountaggregate.AccountAggregate{},
		&paymentaggregate.PaymentBalanceAggregate{},
		&paymentaggregate.PaymentTransactionAggregate{},
	}

	for _, agg := range aggregates {
		if err := serializer.RegisterAggregate(agg); err != nil {
			return nil, stackerr.Error(err)
		}
	}

	return serializer, nil
}

func decodeEventPayload(ctx context.Context, serializer eventpkg.Serializer, aggregateType, eventName string, raw []byte) (interface{}, error) {
	logger := logging.FromContext(ctx)
	if serializer == nil {
		return nil, stackerr.Error(fmt.Errorf("event serializer is nil"))
	}

	payloadFactory, ok := serializer.Type(aggregateType, eventName)
	if !ok {
		logger.Warnw("unsupported event_name",
			zap.String("aggregate_type", aggregateType),
			zap.String("event_name", eventName),
		)
		return nil, nil
	}

	payload := clonePayload(payloadFactory())
	if payload == nil {
		return nil, stackerr.Error(fmt.Errorf("event payload prototype is nil"))
	}

	if err := unmarshalEventData(raw, payload); err != nil {
		logger.Errorw("unmarshal event_data failed", zap.Error(err), zap.String("raw", string(raw)))
		return nil, stackerr.Error(fmt.Errorf("unmarshal event_data failed: %w", err))
	}

	return payload, nil
}

func clonePayload(prototype interface{}) interface{} {
	payloadType := reflect.TypeOf(prototype)
	if payloadType == nil {
		return nil
	}
	if payloadType.Kind() == reflect.Ptr {
		return reflect.New(payloadType.Elem()).Interface()
	}
	return reflect.New(payloadType).Interface()
}

func unmarshalEventData(raw []byte, target interface{}) error {
	if len(raw) == 0 {
		return stackerr.Error(fmt.Errorf("event_data is empty"))
	}

	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return stackerr.Error(fmt.Errorf("unmarshal event_data as string failed: %w", err))
	}

	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return stackerr.Error(fmt.Errorf("event_data is empty"))
	}

	if err := json.Unmarshal([]byte(encoded), target); err != nil {
		return stackerr.Error(fmt.Errorf("unmarshal encoded event_data failed: %w", err))
	}

	return nil
}
