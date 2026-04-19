// CODE_GENERATOR - do not edit: response
package out

type ListNotificationResponse struct {
	Notifications []NotificationResponse `json:"notifications,omitempty"`
	NextCursor    string                 `json:"next_cursor,omitempty"`
	HasMore       bool                   `json:"has_more,omitempty"`
	Total         int                    `json:"total,omitempty"`
	Limit         int                    `json:"limit,omitempty"`
	UnreadCount   int                    `json:"unread_count,omitempty"`
}

type NotificationResponse struct {
	ID                 string `json:"id,omitempty"`
	AccountID          string `json:"account_id,omitempty"`
	Kind               string `json:"kind,omitempty"`
	Type               string `json:"type,omitempty"`
	GroupKey           string `json:"group_key,omitempty"`
	Subject            string `json:"subject,omitempty"`
	Body               string `json:"body,omitempty"`
	IsRead             bool   `json:"is_read,omitempty"`
	ReadAt             string `json:"read_at,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
	SortAt             string `json:"sort_at,omitempty"`
	RoomID             string `json:"room_id,omitempty"`
	RoomName           string `json:"room_name,omitempty"`
	SenderID           string `json:"sender_id,omitempty"`
	SenderName         string `json:"sender_name,omitempty"`
	MessageCount       int    `json:"message_count,omitempty"`
	LastMessageID      string `json:"last_message_id,omitempty"`
	LastMessagePreview string `json:"last_message_preview,omitempty"`
	LastMessageAt      string `json:"last_message_at,omitempty"`
}
