// CODE_GENERATOR: request

package in

type ListChatConversationsRequest struct {
	Limit  int `json:"limit" form:"limit"`
	Offset int `json:"offset" form:"offset"`
}

func (r *ListChatConversationsRequest) Validate() error {
	return nil
}
