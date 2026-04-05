package in

type ListChatConversationsRequest struct {
	Limit  int `form:"limit"`
	Offset int `form:"offset"`
}

func (r *ListChatConversationsRequest) Validate() error {
	if r.Limit < 0 {
		r.Limit = 0
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	return nil
}
