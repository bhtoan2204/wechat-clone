package valueobject

import "go-socket/core/shared/pkg/stackErr"

type PlainPassword struct {
	value string
}

func NewPlainPassword(value string) (PlainPassword, error) {
	normalized, err := normalizePasswordValue(value)
	if err != nil {
		return PlainPassword{}, stackErr.Error(err)
	}
	return PlainPassword{value: normalized}, nil
}

func (p PlainPassword) Value() string {
	return p.value
}
