// CODE_GENERATOR: response
package out

type ListNotificationResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	NextCursor    string                 `json:"next_cursor"`
	HasMore       bool                   `json:"has_more"`
	Total         int                    `json:"total"`
	Limit         int                    `json:"limit"`
}

type NotificationResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Type      string `json:"type"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	IsRead    bool   `json:"is_read"`
	ReadAt    string `json:"read_at"`
	CreatedAt string `json:"created_at"`
}
