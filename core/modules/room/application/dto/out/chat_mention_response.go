package out

type ChatMessageMentionResponse struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

type ChatMentionCandidateResponse struct {
	AccountID       string `json:"account_id"`
	DisplayName     string `json:"display_name"`
	Username        string `json:"username"`
	AvatarObjectKey string `json:"avatar_object_key"`
}
