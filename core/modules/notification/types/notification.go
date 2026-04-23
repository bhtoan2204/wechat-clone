package types

import "strings"

type NotificationKind string

const (
	NotificationKindGeneral NotificationKind = "general"
	NotificationKindMessage NotificationKind = "message"
)

func (k NotificationKind) String() string {
	return string(k)
}

func (k NotificationKind) Normalize() NotificationKind {
	return NotificationKind(strings.ToLower(strings.TrimSpace(string(k))))
}

type NotificationType string

const (
	NotificationTypeAccountCreated         NotificationType = "account.created"
	NotificationTypeRoomMention            NotificationType = "room.mention"
	NotificationTypeRoomMessage            NotificationType = "room.message"
	NotificationTypeFriendRequestSent      NotificationType = "relationship.friend_request.sent"
	NotificationTypeFriendRequestCancelled NotificationType = "relationship.friend_request.cancelled"
	NotificationTypeFriendRequestAccepted  NotificationType = "relationship.friend_request.accepted"
	NotificationTypeFriendRequestRejected  NotificationType = "relationship.friend_request.rejected"
)

func (t NotificationType) String() string {
	return string(t)
}

func (t NotificationType) Normalize() NotificationType {
	return NotificationType(strings.ToLower(strings.TrimSpace(string(t))))
}

const (
	RealtimeEventNotificationUpsert  = "notification.upsert"
	RealtimeEventNotificationRead    = "notification.read"
	RealtimeEventNotificationReadAll = "notification.read_all"
	RealtimeEventUnreadCountUpdated  = "notification.unread_count.updated"
	MessageNotificationGroupPrefix   = "room:"
)

type RealtimeMessagePayload struct {
	RoomID  string
	Type    string
	Payload interface{}
}
