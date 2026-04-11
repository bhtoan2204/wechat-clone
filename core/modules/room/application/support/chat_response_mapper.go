package support

import (
	"go-socket/core/modules/room/application/dto/out"
	apptypes "go-socket/core/modules/room/application/types"
)

func ToConversationResponse(res *apptypes.ConversationResult) *out.ChatConversationResponse {
	if res == nil {
		return nil
	}

	members := make([]out.ChatRoomMemberResponse, 0, len(res.Members))
	for _, member := range res.Members {
		members = append(members, out.ChatRoomMemberResponse{
			AccountID:       member.AccountID,
			Role:            member.Role,
			DisplayName:     member.DisplayName,
			AvatarObjectKey: member.AvatarObjectKey,
		})
	}

	return &out.ChatConversationResponse{
		RoomID:          res.RoomID,
		Name:            res.Name,
		Description:     res.Description,
		RoomType:        res.RoomType,
		OwnerID:         res.OwnerID,
		PinnedMessageID: res.PinnedMessageID,
		MemberCount:     res.MemberCount,
		UnreadCount:     res.UnreadCount,
		LastMessage:     ToMessageResponse(res.LastMessage),
		Members:         members,
		CreatedAt:       res.CreatedAt,
		UpdatedAt:       res.UpdatedAt,
	}
}

func ToMessageResponse(res *apptypes.MessageResult) *out.ChatMessageResponse {
	if res == nil {
		return nil
	}

	mentions := make([]out.ChatMessageMentionResponse, 0, len(res.Mentions))
	for _, mention := range res.Mentions {
		mentions = append(mentions, out.ChatMessageMentionResponse{
			AccountID:   mention.AccountID,
			DisplayName: mention.DisplayName,
			Username:    mention.Username,
		})
	}

	return &out.ChatMessageResponse{
		ID:                     res.ID,
		RoomID:                 res.RoomID,
		SenderID:               res.SenderID,
		Message:                res.Message,
		MessageType:            res.MessageType,
		Status:                 res.Status,
		Mentions:               mentions,
		MentionAll:             res.MentionAll,
		ReplyToMessageID:       res.ReplyToMessageID,
		ForwardedFromMessageID: res.ForwardedFromMessageID,
		FileName:               res.FileName,
		FileSize:               res.FileSize,
		MimeType:               res.MimeType,
		ObjectKey:              res.ObjectKey,
		EditedAt:               res.EditedAt,
		DeletedForEveryone:     res.DeletedForEveryone,
		CreatedAt:              res.CreatedAt,
		ReplyTo:                toPreviewResponse(res.ReplyTo),
		ForwardedFrom:          toPreviewResponse(res.ForwardedFrom),
	}
}

func ToPresenceResponse(res *apptypes.PresenceResult) *out.ChatPresenceResponse {
	if res == nil {
		return nil
	}
	return &out.ChatPresenceResponse{
		AccountID: res.AccountID,
		Status:    res.Status,
	}
}

func toPreviewResponse(res *apptypes.MessagePreviewResult) *out.ChatMessagePreviewResponse {
	if res == nil {
		return nil
	}
	return &out.ChatMessagePreviewResponse{
		ID:          res.ID,
		SenderID:    res.SenderID,
		Message:     res.Message,
		MessageType: res.MessageType,
	}
}
