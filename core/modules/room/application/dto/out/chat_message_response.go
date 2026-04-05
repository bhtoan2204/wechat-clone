package out

type ChatMessageResponse struct {
	ID                     string                      `json:"id"`
	RoomID                 string                      `json:"room_id"`
	SenderID               string                      `json:"sender_id"`
	Message                string                      `json:"message"`
	MessageType            string                      `json:"message_type"`
	Status                 string                      `json:"status"`
	ReplyToMessageID       string                      `json:"reply_to_message_id,omitempty"`
	ForwardedFromMessageID string                      `json:"forwarded_from_message_id,omitempty"`
	FileName               string                      `json:"file_name,omitempty"`
	FileSize               int64                       `json:"file_size,omitempty"`
	MimeType               string                      `json:"mime_type,omitempty"`
	ObjectKey              string                      `json:"object_key,omitempty"`
	EditedAt               string                      `json:"edited_at,omitempty"`
	DeletedForEveryone     bool                        `json:"deleted_for_everyone"`
	CreatedAt              string                      `json:"created_at"`
	ReplyTo                *ChatMessagePreviewResponse `json:"reply_to,omitempty"`
	ForwardedFrom          *ChatMessagePreviewResponse `json:"forwarded_from,omitempty"`
}
