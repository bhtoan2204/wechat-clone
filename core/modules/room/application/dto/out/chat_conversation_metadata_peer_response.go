package out

type ChatConversationMetadataPeerResponse struct {
	AccountID       string `json:"account_id,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	Username        string `json:"username,omitempty"`
	AvatarObjectKey string `json:"avatar_object_key,omitempty"`
}
