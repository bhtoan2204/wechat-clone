package valueobject

import (
	sharedevents "go-socket/core/shared/contracts/events"
)

type MessageMentions struct {
	Items      []sharedevents.RoomMessageMention
	MentionAll bool
	AccountIDs []string
}
