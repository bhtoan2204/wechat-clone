package support

import (
	"wechat-clone/core/modules/room/application/dto/out"
	apptypes "wechat-clone/core/modules/room/application/types"

	"github.com/samber/lo"
)

func ToConversationResponse(res *apptypes.ConversationResult) *out.ChatConversationResponse {
	if res == nil {
		return nil
	}

	members := lo.Map(res.Members, func(member apptypes.ConversationMemberResult, _ int) out.ChatRoomMemberResponse {
		return out.ChatRoomMemberResponse{
			AccountID:       member.AccountID,
			Role:            member.Role,
			DisplayName:     member.DisplayName,
			AvatarObjectKey: member.AvatarObjectKey,
		}
	})

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

func ToConversationMetadataResponse(res *apptypes.ConversationMetadataResult) *out.ChatConversationMetadataResponse {
	if res == nil {
		return nil
	}

	var peer *out.ChatConversationMetadataPeerResponse
	if res.DirectPeer != nil {
		peer = &out.ChatConversationMetadataPeerResponse{
			AccountID:       res.DirectPeer.AccountID,
			DisplayName:     res.DirectPeer.DisplayName,
			Username:        res.DirectPeer.Username,
			AvatarObjectKey: res.DirectPeer.AvatarObjectKey,
		}
	}

	return &out.ChatConversationMetadataResponse{
		RoomID:                res.RoomID,
		RoomType:              res.RoomType,
		OwnerID:               res.OwnerID,
		MemberCount:           res.MemberCount,
		PinnedMessageID:       res.PinnedMessageID,
		LastMessageID:         res.LastMessageID,
		ViewerRole:            res.ViewerRole,
		ViewerLastDeliveredAt: res.ViewerLastDeliveredAt,
		ViewerLastReadAt:      res.ViewerLastReadAt,
		IsOwner:               res.IsOwner,
		DirectPeer:            peer,
	}
}

func ToMessageResponse(res *apptypes.MessageResult) *out.ChatMessageResponse {
	if res == nil {
		return nil
	}

	mentions := lo.Map(res.Mentions, func(mention apptypes.MessageMentionResult, _ int) out.ChatMessageMentionResponse {
		return out.ChatMessageMentionResponse{
			AccountID:   mention.AccountID,
			DisplayName: mention.DisplayName,
			Username:    mention.Username,
		}
	})
	reactions := lo.Map(res.Reactions, func(item apptypes.MessageReactionResult, _ int) out.ChatMessageReactionResponse {
		return out.ChatMessageReactionResponse{
			Emoji:       item.Emoji,
			Count:       item.Count,
			ReactedByMe: item.ReactedByMe,
			AccountIDs:  item.AccountIDs,
		}
	})

	return &out.ChatMessageResponse{
		ID:                     res.ID,
		RoomID:                 res.RoomID,
		SenderID:               res.SenderID,
		Message:                res.Message,
		MessageType:            res.MessageType,
		Status:                 res.Status,
		Mentions:               mentions,
		Reactions:              reactions,
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
