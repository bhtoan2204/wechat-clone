package valueobject

type MessageReference struct {
	ReplyToMessageID       string
	ForwardedFromMessageID string
}

func (m MessageReference) IsReply() bool { return m.ReplyToMessageID != "" }
