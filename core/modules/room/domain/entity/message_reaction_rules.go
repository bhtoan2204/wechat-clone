package entity

import (
	"strings"
	"time"

	"wechat-clone/core/shared/pkg/stackErr"
)

func (m *MessageEntity) ToggleReaction(accountID, emoji string, reactedAt time.Time) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	emoji = strings.TrimSpace(emoji)

	switch {
	case accountID == "":
		return false, stackErr.Error(ErrMessageReactionAccountIDRequired)
	case emoji == "":
		return false, stackErr.Error(ErrMessageReactionEmojiRequired)
	}

	current, err := NormalizeMessageReactions(m.Reactions)
	if err != nil {
		return false, stackErr.Error(err)
	}

	removed := false
	next := make([]MessageReaction, 0, len(current)+1)
	for _, item := range current {
		if item.AccountID == accountID && item.Emoji == emoji {
			removed = true
			continue
		}
		next = append(next, item)
	}

	if !removed {
		next = append(next, MessageReaction{
			AccountID: accountID,
			Emoji:     emoji,
			ReactedAt: normalizeRoomTime(reactedAt),
		})
	}

	next, err = NormalizeMessageReactions(next)
	if err != nil {
		return false, stackErr.Error(err)
	}
	m.Reactions = next
	return true, nil
}
