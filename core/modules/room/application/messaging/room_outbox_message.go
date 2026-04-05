package messaging

import (
	"encoding/json"
	"strings"
)

type roomOutboxMessage struct {
	ID            int64           `json:"id"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int64           `json:"version"`
	EventName     string          `json:"event_name"`
	EventData     json.RawMessage `json:"event_data"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     string          `json:"created_at"`
}

func (m *roomOutboxMessage) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
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

	type alias roomOutboxMessage
	var aux alias
	normalizedData, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(normalizedData, &aux); err != nil {
		return err
	}

	*m = roomOutboxMessage(aux)
	return nil
}
