package repository

import (
	"context"

	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type messageAggregateRepoImpl struct {
	db             *gorm.DB
	messageRepo    repos.MessageRepository
	roomRepo       repos.RoomRepository
	roomMemberRepo repos.RoomMemberRepository
	accountRepo    repos.RoomAccountProjectionRepository
	outboxRepo     repos.RoomOutboxEventsRepository
}

func newMessageAggregateRepoImpl(
	db *gorm.DB,
	messageRepo repos.MessageRepository,
	roomRepo repos.RoomRepository,
	roomMemberRepo repos.RoomMemberRepository,
	accountRepo repos.RoomAccountProjectionRepository,
	outboxRepo repos.RoomOutboxEventsRepository,
) repos.MessageAggregateRepository {
	return &messageAggregateRepoImpl{
		db:             db,
		messageRepo:    messageRepo,
		roomRepo:       roomRepo,
		roomMemberRepo: roomMemberRepo,
		accountRepo:    accountRepo,
		outboxRepo:     outboxRepo,
	}
}

func (r *messageAggregateRepoImpl) Load(ctx context.Context, messageID string) (*aggregate.MessageStateAggregate, error) {
	message, err := r.messageRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return aggregate.NewMessageStateAggregate(message)
}

func (r *messageAggregateRepoImpl) Save(ctx context.Context, agg *aggregate.MessageStateAggregate) error {
	if agg == nil {
		return stackErr.Error(aggregate.ErrMessageAggregateNil)
	}

	roomID := agg.Message().RoomID
	pendingOutboxEvents := make([]pendingRoomOutboxEvent, 0, 2)

	if agg.MessageDirty() {
		if err := r.messageRepo.UpdateMessage(ctx, agg.Message()); err != nil {
			return stackErr.Error(err)
		}
	}

	var updatedMembers []*entity.RoomMemberEntity
	if agg.MemberDirty() && agg.RecipientMember() != nil {
		if err := r.roomMemberRepo.UpdateRoomMember(ctx, agg.RecipientMember()); err != nil {
			return stackErr.Error(err)
		}
		updatedMembers = []*entity.RoomMemberEntity{agg.RecipientMember()}
	}

	hasProjectionChange := agg.MessageDirty() || len(updatedMembers) > 0 || agg.PendingReceipt() != nil || agg.PendingDeletion() != nil
	if hasProjectionChange {
		room, err := r.roomRepo.GetRoomByID(ctx, roomID)
		if err != nil {
			return stackErr.Error(err)
		}

		updatedMembers, err = enrichRoomMembersWithAccountProjections(ctx, r.accountRepo, updatedMembers)
		if err != nil {
			return stackErr.Error(err)
		}

		var senderProjection *entity.AccountEntity
		if r.accountRepo != nil {
			accountProjections, projectionErr := r.accountRepo.ListByAccountIDs(ctx, []string{agg.Message().SenderID})
			if projectionErr != nil {
				return stackErr.Error(projectionErr)
			}
			if len(accountProjections) > 0 {
				senderProjection = accountProjections[0]
			}
		}

		senderName, senderEmail := senderIdentityFromProjection(senderProjection, agg.Message().SenderID)

		receipts := make([]aggregate.PendingMessageReceipt, 0, 1)
		if receipt := agg.PendingReceipt(); receipt != nil {
			receipts = append(receipts, *receipt)
		}

		deletions := make([]*aggregate.PendingMessageDeletion, 0, 1)
		if deletion := agg.PendingDeletion(); deletion != nil {
			deletions = append(deletions, deletion)
		}

		pendingOutboxEvents = append(pendingOutboxEvents, buildMessageAggregateProjectionSyncEvent(
			agg.Message(),
			room,
			senderName,
			senderEmail,
			updatedMembers,
			receipts,
			deletions,
		))

		if agg.MessageDirty() {
			members, err := r.roomMemberRepo.ListRoomMembers(ctx, roomID)
			if err != nil {
				return stackErr.Error(err)
			}
			members, err = enrichRoomMembersWithAccountProjections(ctx, r.accountRepo, members)
			if err != nil {
				return stackErr.Error(err)
			}

			lastMessage, err := r.messageRepo.GetLastMessageByRoomID(ctx, roomID)
			if err != nil {
				return stackErr.Error(err)
			}

			pendingOutboxEvents = append(pendingOutboxEvents, buildRoomAggregateProjectionSyncEvent(
				room,
				sortRoomMembersByAccount(members),
				lastMessage,
			))
		}
	}

	if len(pendingOutboxEvents) > 0 {
		baseVersion, err := loadLatestRoomOutboxVersion(ctx, r.db, roomID)
		if err != nil {
			return stackErr.Error(err)
		}
		if _, err := appendRoomOutboxEvents(ctx, r.outboxRepo, roomID, baseVersion, pendingOutboxEvents); err != nil {
			return stackErr.Error(err)
		}
	}

	agg.MarkPersisted()
	return nil
}
