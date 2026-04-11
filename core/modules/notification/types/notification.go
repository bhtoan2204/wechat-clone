package types

type NotificationType string

const (
	NotificationTypeAccountCreated NotificationType = "account.created"
	NotificationTypeRoomMention    NotificationType = "room.mention"
)

func (t NotificationType) String() string {
	return string(t)
}
