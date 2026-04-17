package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"reflect"
	"strings"

	"go.uber.org/zap"
)

var eventPayloadTypes = map[string]reflect.Type{
	sharedevents.EventAccountCreated:        reflect.TypeOf(sharedevents.AccountCreatedEvent{}),
	sharedevents.EventAccountProfileUpdated: reflect.TypeOf(sharedevents.AccountProfileUpdatedEvent{}),

	sharedevents.EventLedgerAccountTransferredToAccount: reflect.TypeOf(sharedevents.LedgerTransaction{}),
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
