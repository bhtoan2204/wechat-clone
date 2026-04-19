package models

type TableNames struct {
	NotificationByID          string
	NotificationByAccount     string
	NotificationUnreadIndex   string
	MessageNotificationGroups string
	PushSubscriptions         string
	SchemaMigrations          string
}

func DefaultTableNames() TableNames {
	return TableNames{
		NotificationByID:          "notifications_by_id",
		NotificationByAccount:     "notifications_by_account",
		NotificationUnreadIndex:   "notification_unread_by_account",
		MessageNotificationGroups: "message_notification_groups_by_account",
		PushSubscriptions:         "push_subscriptions_by_account",
		SchemaMigrations:          "notification_schema_migrations",
	}
}
