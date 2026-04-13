package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

const roomOutboxAggregateType = "RoomAggregate"

type roomAggregateRepoImpl struct {
	db              *gorm.DB
	roomRepo        repos.RoomRepository
	roomMemberRepo  repos.RoomMemberRepository
	messageRepo     repos.MessageRepository
	outboxRepo      repos.RoomOutboxEventsRepository
	roomAccountRepo repos.RoomAccountProjectionRepository
}

func newRoomAggregateRepoImpl(db *gorm.DB,
	roomRepo repos.RoomRepository,
	roomMemberRepo repos.RoomMemberRepository,
	messageRepo repos.MessageRepository,
	outboxRepo repos.RoomOutboxEventsRepository,
	accountRepo repos.RoomAccountProjectionRepository,
) repos.RoomAggregateRepository {
	return &roomAggregateRepoImpl{
		db:              db,
		roomRepo:        roomRepo,
		roomMemberRepo:  roomMemberRepo,
		messageRepo:     messageRepo,
		roomAccountRepo: accountRepo,
		outboxRepo:      outboxRepo,
	}
}

func (r *roomAggregateRepoImpl) Load(ctx context.Context, roomID string) (*aggregate.RoomStateAggregate, error) {
	room, err := r.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	members, err := r.roomMemberRepo.ListRoomMembers(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountProjections, err := r.roomAccountRepo.ListByAccountIDs(ctx, lo.Map(members, func(member *entity.RoomMemberEntity, _ int) string {
		return member.AccountID
	}))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	accountMap := lo.KeyBy(accountProjections, func(acc *entity.AccountEntity) string {
		return acc.AccountID
	})

	members = lo.Map(members, func(member *entity.RoomMemberEntity, _ int) *entity.RoomMemberEntity {
		if acc, exists := accountMap[member.AccountID]; exists {
			member.DisplayName = acc.DisplayName
			member.Username = acc.Username
			member.AvatarObjectKey = acc.AvatarObjectKey
		}

		return member
	})

	version, err := r.loadLatestOutboxVersion(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return aggregate.RestoreRoomStateAggregate(room, members, version)
}

func (r *roomAggregateRepoImpl) LoadByDirectKey(ctx context.Context, directKey string) (*aggregate.RoomStateAggregate, error) {
	room, err := r.roomRepo.GetRoomByDirectKey(ctx, directKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return r.Load(ctx, room.ID)
}

func (r *roomAggregateRepoImpl) Save(ctx context.Context, agg *aggregate.RoomStateAggregate) error {
	if agg == nil {
		return stackErr.Error(aggregate.ErrRoomAggregateNil)
	}
	if agg.IsDeleted() {
		return stackErr.Error(errors.New("deleted room aggregate must be removed via Delete"))
	}
	if !agg.HasPendingRoomWrite() {
		return nil
	}

	room := agg.Room()
	if room == nil {
		return stackErr.Error(aggregate.ErrRoomAggregateNil)
	}

	if agg.IsNew() {
		if err := r.roomRepo.CreateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
	} else {
		if err := r.roomRepo.UpdateRoom(ctx, room); err != nil {
			return stackErr.Error(err)
		}
	}

	for _, memberID := range agg.RemovedMemberIDs() {
		if err := r.roomMemberRepo.DeleteRoomMember(ctx, room.ID, memberID); err != nil {
			return stackErr.Error(err)
		}
	}

	pendingMemberUpserts := agg.PendingMemberUpserts()
	sort.Slice(pendingMemberUpserts, func(i, j int) bool {
		if pendingMemberUpserts[i] == nil || pendingMemberUpserts[j] == nil {
			return i < j
		}
		return pendingMemberUpserts[i].AccountID < pendingMemberUpserts[j].AccountID
	})
	for _, member := range pendingMemberUpserts {
		if member == nil {
			continue
		}
		existing, err := r.roomMemberRepo.GetRoomMemberByAccount(ctx, member.RoomID, member.AccountID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(err)
			}
			if err := r.roomMemberRepo.CreateRoomMember(ctx, member); err != nil {
				return stackErr.Error(err)
			}
		} else if existing != nil {
			if err := r.roomMemberRepo.UpdateRoomMember(ctx, member); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	pendingMessages := agg.PendingMessages()
	for _, message := range pendingMessages {
		if message == nil {
			continue
		}
		if err := r.messageRepo.CreateMessage(ctx, message); err != nil {
			return stackErr.Error(err)
		}
	}

	pendingOutboxEvents := make([]pendingRoomOutboxEvent, 0, len(agg.RemovedMemberIDs())+len(pendingMemberUpserts)+len(pendingMessages)+len(agg.PendingReceipts())+len(agg.PendingOutboxEvents())+1)

	for _, memberID := range agg.RemovedMemberIDs() {
		pendingOutboxEvents = append(pendingOutboxEvents, buildRoomMemberProjectionDeleteEvent(room.ID, memberID, room.UpdatedAt))
	}
	for _, member := range pendingMemberUpserts {
		if member == nil {
			continue
		}
		pendingOutboxEvents = append(pendingOutboxEvents, buildRoomMemberProjectionUpsertEvent(member))
	}

	domainPendingEvents := agg.PendingOutboxEvents()
	for _, message := range pendingMessages {
		if message == nil {
			continue
		}
		senderName, senderEmail := senderIdentityFromPendingEvents(message.ID, domainPendingEvents)
		if senderName == "" {
			senderName = message.SenderID
		}
		pendingOutboxEvents = append(pendingOutboxEvents, buildRoomMessageProjectionUpsertEvent(message, room, senderName, senderEmail))
	}
	for _, receipt := range agg.PendingReceipts() {
		pendingOutboxEvents = append(pendingOutboxEvents, buildRoomMessageReceiptProjectionEvent(room.ID, receipt))
	}

	var lastPendingMessage *entity.MessageEntity
	if len(pendingMessages) > 0 {
		lastPendingMessage = pendingMessages[len(pendingMessages)-1]
	}
	pendingOutboxEvents = append(
		pendingOutboxEvents,
		buildRoomProjectionUpsertEvent(room, len(agg.Members()), lastPendingMessage, agg.IsNew() || len(pendingMessages) > 0),
	)

	for _, pendingEvent := range domainPendingEvents {
		pendingOutboxEvents = append(pendingOutboxEvents, pendingRoomOutboxEvent{
			EventName: pendingEvent.EventName,
			Payload:   pendingEvent.Payload,
			CreatedAt: pendingEvent.CreatedAt,
		})
	}

	nextVersion, err := appendRoomOutboxEvents(ctx, r.outboxRepo, room.ID, agg.BaseVersion(), pendingOutboxEvents)
	if err != nil {
		return stackErr.Error(err)
	}

	agg.MarkPersisted(nextVersion)
	return nil
}

func (r *roomAggregateRepoImpl) Delete(ctx context.Context, roomID string) error {
	if err := r.roomRepo.DeleteRoom(ctx, roomID); err != nil {
		return stackErr.Error(err)
	}

	baseVersion, err := r.loadLatestOutboxVersion(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}

	_, err = appendRoomOutboxEvents(ctx, r.outboxRepo, roomID, baseVersion, []pendingRoomOutboxEvent{
		buildRoomProjectionDeleteEvent(roomID, time.Now().UTC()),
	})
	return stackErr.Error(err)
}

func (r *roomAggregateRepoImpl) loadLatestOutboxVersion(ctx context.Context, roomID string) (int, error) {
	return loadLatestRoomOutboxVersion(ctx, r.db, roomID)
}
