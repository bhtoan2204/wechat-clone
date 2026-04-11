package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

var projectionEventPayloadTypes = map[string]reflect.Type{
	sharedevents.EventRoomMessageCreated: reflect.TypeOf(sharedevents.RoomMessageCreatedEvent{}),
}

func decodeEventPayload(ctx context.Context, eventName string, raw []byte) (interface{}, error) {
	logger := logging.FromContext(ctx)
	payloadType, ok := projectionEventPayloadTypes[eventName]
	if !ok {
		logger.Warnw("unsupported event_name", zap.String("event_name", eventName))
		return nil, nil
	}

	payload := reflect.New(payloadType).Interface()
	if err := unmarshalEventData(raw, payload); err != nil {
		logger.Errorw("unmarshal event_data failed", zap.Error(err), zap.String("raw", string(raw)))
		return nil, stackErr.Error(fmt.Errorf("unmarshal event_data failed: %v", err))
	}

	return payload, nil
}

func unmarshalEventData(raw []byte, target interface{}) error {
	if len(raw) == 0 {
		return stackErr.Error(fmt.Errorf("event_data is empty"))
	}

	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal event_data as string failed: %v", err))
	}

	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return stackErr.Error(fmt.Errorf("event_data is empty"))
	}

	if err := json.Unmarshal([]byte(encoded), target); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal encoded event_data failed: %v", err))
	}

	return nil
}
