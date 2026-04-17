package command

import (
	"context"
	"reflect"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type sendChatMessageHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewSendChatMessageHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.SendChatMessageRequest, *out.ChatMessageResponse] {
	return &sendChatMessageHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *sendChatMessageHandler) Handle(ctx context.Context, req *in.SendChatMessageRequest) (*out.ChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := executeSendMessage(ctx, h.baseRepo, accountID, apptypes.SendMessageCommand{
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

	out := roomsupport.ToMessageResponse(res)
	h.realtime.EmitMessage(ctx, types.MessagePayload{
		RoomId:  out.RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	})
	return out, nil
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
