package valueobject

import (
	"encoding/json"

	"go-socket/core/shared/pkg/stackErr"
)

type HashedPassword struct {
	value string
}

func NewHashedPassword(value string) (HashedPassword, error) {
	normalized, err := normalizePasswordValue(value)
	if err != nil {
		return HashedPassword{}, stackErr.Error(err)
	}
	return HashedPassword{value: normalized}, nil
}

func (p HashedPassword) Value() string {
	return p.value
}

func (p HashedPassword) IsZero() bool {
	return p.value == ""
}

func (p HashedPassword) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

func (p *HashedPassword) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return stackErr.Error(err)
	}

	password, err := NewHashedPassword(value)
	if err != nil {
		return stackErr.Error(err)
	}

	*p = password
	return nil
}
