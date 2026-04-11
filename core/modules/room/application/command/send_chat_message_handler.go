package command

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type sendChatMessageHandler struct {
	messageService *roomservice.MessageCommandService
}

func NewSendChatMessageHandler(messageService *roomservice.MessageCommandService) cqrs.Handler[*in.SendChatMessageRequest, *out.ChatMessageResponse] {
	return &sendChatMessageHandler{messageService: messageService}
}

func (h *sendChatMessageHandler) Handle(ctx context.Context, req *in.SendChatMessageRequest) (*out.ChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := h.messageService.SendMessage(ctx, accountID, apptypes.SendMessageCommand{
		RoomID:                 req.RoomID,
		Message:                req.Message,
		MessageType:            req.MessageType,
		Mentions:               mapMentionCommands(req.Mentions),
		MentionAll:             req.MentionAll,
		ReplyToMessageID:       req.ReplyToMessageID,
		ForwardedFromMessageID: req.ForwardedFromMessageID,
		FileName:               req.FileName,
		FileSize:               req.FileSize,
		MimeType:               req.MimeType,
		ObjectKey:              req.ObjectKey,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return roomsupport.ToMessageResponse(res), nil
}

func mapMentionCommands(items []in.SendChatMessageMentionRequest) []apptypes.SendMessageMentionCommand {
	if len(items) == 0 {
		return nil
	}

	results := make([]apptypes.SendMessageMentionCommand, 0, len(items))
	for _, item := range items {
		results = append(results, apptypes.SendMessageMentionCommand{
			AccountID: item.AccountID,
		})
	}
	return results
}
