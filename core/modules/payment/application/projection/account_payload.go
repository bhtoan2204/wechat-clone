package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"reflect"
	"strings"

	"go.uber.org/zap"
)

var eventPayloadTypes = map[string]reflect.Type{
	"EventAccountCreated": reflect.TypeOf(aggregate.EventAccountCreated{}),
	"EventAccountUpdated": reflect.TypeOf(aggregate.EventAccountUpdated{}),
	"EventAccountBanned":  reflect.TypeOf(aggregate.EventAccountBanned{}),
}

func decodeEventPayload(ctx context.Context, eventName string, raw []byte) (interface{}, error) {
	logger := logging.FromContext(ctx)
	payloadType, ok := eventPayloadTypes[eventName]
	if !ok {
		logger.Warnw("unsupported event_name", zap.String("event_name", eventName))
		return nil, nil
	}

	payload := reflect.New(payloadType).Interface()
	if err := unmarshalEventData(raw, payload); err != nil {
		logger.Errorw("unmarshal event_data failed", zap.Error(err), zap.String("raw", string(raw)))
		return nil, stackerr.Error(fmt.Errorf("unmarshal event_data failed: %w", err))
	}

	return payload, nil
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
