package contracts

import (
	"encoding/json"
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type OutboxMessage struct {
	ID            json.RawMessage `json:"id"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int64           `json:"version"`
	EventName     string          `json:"event_name"`
	EventData     json.RawMessage `json:"event_data"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     string          `json:"created_at"`
}

func (m *OutboxMessage) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return stackErr.Error(err)
	}

	normalized := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		lowerKey := strings.ToLower(key)
		if key == lowerKey {
			normalized[lowerKey] = value
		}
	}
	for key, value := range raw {
		lowerKey := strings.ToLower(key)
		if _, exists := normalized[lowerKey]; !exists {
			normalized[lowerKey] = value
		}
	}

	type alias OutboxMessage
	var aux alias
	normalizedData, err := json.Marshal(normalized)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := json.Unmarshal(normalizedData, &aux); err != nil {
		return stackErr.Error(err)
	}

	*m = OutboxMessage(aux)
	return nil
}

func UnmarshalEventData(raw []byte, target interface{}) error {
	if target == nil {
		return stackErr.Error(errors.New("event_data target is required"))
	}
	if len(raw) == 0 {
		return stackErr.Error(errors.New("event_data is empty"))
	}

	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return stackErr.Error(err)
	}

	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return stackErr.Error(errors.New("event_data is empty"))
	}

	return stackErr.Error(json.Unmarshal([]byte(encoded), target))
}
