package out

type ChatMessagePreviewResponse struct {
	ID          string `json:"id"`
	SenderID    string `json:"sender_id"`
	Message     string `json:"message"`
	MessageType string `json:"message_type"`
}
