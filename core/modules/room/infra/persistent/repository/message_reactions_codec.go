package repository

import (
	"encoding/json"
	"strings"

	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/shared/pkg/stackErr"
)

func marshalMessageReactions(items []entity.MessageReaction) (string, error) {
	if len(items) == 0 {
		return "[]", nil
	}

	data, err := json.Marshal(items)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return string(data), nil
}

func unmarshalMessageReactions(raw string) ([]entity.MessageReaction, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var items []entity.MessageReaction
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, stackErr.Error(err)
	}
	return entity.NormalizeMessageReactions(items)
}
