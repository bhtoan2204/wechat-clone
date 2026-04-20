// CODE_GENERATOR - do not edit: response
package out

type ChatMessageResponse struct {
	ID                     string                        `json:"id,omitempty"`
	RoomID                 string                        `json:"room_id,omitempty"`
	SenderID               string                        `json:"sender_id,omitempty"`
	Message                string                        `json:"message,omitempty"`
	MessageType            string                        `json:"message_type,omitempty"`
	Status                 string                        `json:"status,omitempty"`
	Mentions               []ChatMessageMentionResponse  `json:"mentions,omitempty"`
	Reactions              []ChatMessageReactionResponse `json:"reactions,omitempty"`
	MentionAll             bool                          `json:"mention_all,omitempty"`
	ReplyToMessageID       string                        `json:"reply_to_message_id,omitempty"`
	ForwardedFromMessageID string                        `json:"forwarded_from_message_id,omitempty"`
	FileName               string                        `json:"file_name,omitempty"`
	FileSize               int64                         `json:"file_size,omitempty"`
	MimeType               string                        `json:"mime_type,omitempty"`
	ObjectKey              string                        `json:"object_key,omitempty"`
	EditedAt               string                        `json:"edited_at,omitempty"`
	DeletedForEveryone     bool                          `json:"deleted_for_everyone,omitempty"`
	CreatedAt              string                        `json:"created_at,omitempty"`
	ReplyTo                *ChatMessagePreviewResponse   `json:"reply_to,omitempty"`
	ForwardedFrom          *ChatMessagePreviewResponse   `json:"forwarded_from,omitempty"`
}

type ChatMessageReactionResponse struct {
	Emoji       string   `json:"emoji,omitempty"`
	Count       int      `json:"count,omitempty"`
	ReactedByMe bool     `json:"reacted_by_me,omitempty"`
	AccountIDs  []string `json:"account_ids,omitempty"`
}
