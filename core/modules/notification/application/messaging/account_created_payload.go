package messaging

import (
	"context"
	"fmt"
	"go-socket/core/shared/contracts"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"reflect"

	"go.uber.org/zap"
)

var eventPayloadTypes = map[string]reflect.Type{
	sharedevents.EventAccountCreated:     reflect.TypeOf(sharedevents.AccountCreatedEvent{}),
	sharedevents.EventRoomMessageCreated: reflect.TypeOf(sharedevents.RoomMessageCreatedEvent{}),
}

func decodeEventPayload(ctx context.Context, eventName string, raw []byte) (interface{}, error) {
	logger := logging.FromContext(ctx)
	payloadType, ok := eventPayloadTypes[eventName]
	if !ok {
		logger.Warnw("unsupported event_name", zap.String("event_name", eventName))
		return nil, nil
	}

	payload := reflect.New(payloadType).Interface()
	if err := contracts.UnmarshalEventData(raw, payload); err != nil {
		logger.Errorw("unmarshal event_data failed", zap.Error(err), zap.String("raw", string(raw)))
		return nil, stackErr.Error(fmt.Errorf("unmarshal event_data failed: %v", err))
	}

	return payload, nil
}
