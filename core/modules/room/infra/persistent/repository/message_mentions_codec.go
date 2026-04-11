package repository

import (
	"encoding/json"
	"strings"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/shared/pkg/stackErr"
)

func marshalMessageMentions(mentions []entity.MessageMention) (string, error) {
	if len(mentions) == 0 {
		return "[]", nil
	}

	data, err := json.Marshal(mentions)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return string(data), nil
}

func unmarshalMessageMentions(raw string) ([]entity.MessageMention, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var mentions []entity.MessageMention
	if err := json.Unmarshal([]byte(raw), &mentions); err != nil {
		return nil, stackErr.Error(err)
	}
	return entity.NormalizeMessageMentions(mentions)
}
