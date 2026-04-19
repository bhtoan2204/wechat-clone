package support

import (
	"wechat-clone/core/modules/notification/application/dto/out"
	"wechat-clone/core/modules/notification/domain/entity"
	notificationtypes "wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/utils"
)

func ToNotificationResponse(notification *entity.NotificationEntity) out.NotificationResponse {
	if notification == nil {
		return out.NotificationResponse{}
	}

	return out.NotificationResponse{
		ID:                 notification.ID,
		AccountID:          notification.AccountID,
		Kind:               notification.Kind.String(),
		Type:               notification.Type.String(),
		GroupKey:           notification.GroupKey,
		Subject:            notification.Subject,
		Body:               notification.Body,
		IsRead:             notification.IsRead,
		ReadAt:             utils.FormatOptionalTime(notification.ReadAt),
		CreatedAt:          notification.CreatedAt.UTC().Format(timeFormat),
		UpdatedAt:          notification.UpdatedAt.UTC().Format(timeFormat),
		SortAt:             notification.SortAt.UTC().Format(timeFormat),
		RoomID:             notification.RoomID,
		RoomName:           notification.RoomName,
		SenderID:           notification.SenderID,
		SenderName:         notification.SenderName,
		MessageCount:       notification.MessageCount,
		LastMessageID:      notification.LastMessageID,
		LastMessagePreview: notification.LastMessagePreview,
		LastMessageAt:      utils.FormatOptionalTime(notification.LastMessageAt),
	}
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

type RealtimeNotificationData struct {
	Event        string                   `json:"event"`
	AccountID    string                   `json:"account_id"`
	Notification *out.NotificationResponse `json:"notification,omitempty"`
	UnreadCount  int                      `json:"unread_count"`
}

func NewRealtimeNotificationPayload(
	event string,
	notification *entity.NotificationEntity,
	unreadCount int,
) notificationtypes.RealtimeMessagePayload {
	accountID := ""
	var mapped *out.NotificationResponse
	if notification != nil {
		response := ToNotificationResponse(notification)
		accountID = response.AccountID
		mapped = &response
	}

	return notificationtypes.RealtimeMessagePayload{
		RoomID: notificationRoomID(accountID),
		Type:   event,
		Payload: RealtimeNotificationData{
			Event:        event,
			AccountID:    accountID,
			Notification: mapped,
			UnreadCount:  unreadCount,
		},
	}
}

func NewRealtimeReadAllPayload(accountID string) notificationtypes.RealtimeMessagePayload {
	return notificationtypes.RealtimeMessagePayload{
		RoomID: notificationRoomID(accountID),
		Type:   notificationtypes.RealtimeEventNotificationReadAll,
		Payload: RealtimeNotificationData{
			Event:       notificationtypes.RealtimeEventNotificationReadAll,
			AccountID:   accountID,
			UnreadCount: 0,
		},
	}
}

func notificationRoomID(accountID string) string {
	return "notification:" + accountID
}
