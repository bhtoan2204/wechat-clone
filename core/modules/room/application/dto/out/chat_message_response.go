// CODE_GENERATOR - do not edit: response
package out

type ChatMessageResponse struct {
	ID                     string                       `json:"id"`
	RoomID                 string                       `json:"room_id"`
	SenderID               string                       `json:"sender_id"`
	Message                string                       `json:"message"`
	MessageType            string                       `json:"message_type"`
	Status                 string                       `json:"status"`
	Mentions               []ChatMessageMentionResponse `json:"mentions"`
	MentionAll             bool                         `json:"mention_all"`
	ReplyToMessageID       string                       `json:"reply_to_message_id"`
	ForwardedFromMessageID string                       `json:"forwarded_from_message_id"`
	FileName               string                       `json:"file_name"`
	FileSize               int64                        `json:"file_size"`
	MimeType               string                       `json:"mime_type"`
	ObjectKey              string                       `json:"object_key"`
	EditedAt               string                       `json:"edited_at"`
	DeletedForEveryone     bool                         `json:"deleted_for_everyone"`
	CreatedAt              string                       `json:"created_at"`
	ReplyTo                *ChatMessagePreviewResponse  `json:"reply_to"`
	ForwardedFrom          *ChatMessagePreviewResponse  `json:"forwarded_from"`
}
