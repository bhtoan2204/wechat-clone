package types

import roomtypes "go-socket/core/modules/room/types"

type CreateDirectConversationCommand struct {
	PeerAccountID string
}

type CreateGroupCommand struct {
	Name        string
	Description string
	MemberIDs   []string
}

type UpdateGroupCommand struct {
	Name        string
	Description string
}

type AddMemberCommand struct {
	AccountID string
	Role      roomtypes.RoomRole
}

type RemoveMemberCommand struct {
	AccountID string
}

type PinMessageCommand struct {
	MessageID string
}

type SendMessageMentionCommand struct {
	AccountID string
}

type SendMessageCommand struct {
	RoomID                 string
	Message                string
	MessageType            string
	Mentions               []SendMessageMentionCommand
	MentionAll             bool
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
}

type EditMessageCommand struct {
	Message string
}

type DeleteMessageCommand struct {
	Scope string
}

type ForwardMessageCommand struct {
	TargetRoomID string
}

type MarkMessageStatusCommand struct {
	Status string
}
