package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	roomprojection "wechat-clone/core/modules/room/application/projection"
	"wechat-clone/core/modules/room/infra/projection/cassandra/read_repo"
	"wechat-clone/core/modules/room/infra/projection/cassandra/views"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"github.com/gocql/gocql"
)

var _ roomprojection.ServingProjector = (*cassandraProjectionStore)(nil)

const (
	globalRoomProjectionPartition      = "all"
	defaultRoomListPageExpansionFactor = 3
)

type cassandraProjectionStore struct {
	session   *gocql.Session
	tables    views.ProjectionTableNames
	rooms     *read_repo.RoomProjectionRepo
	messages  *read_repo.MessageProjectionRepo
	receipts  *read_repo.MessageReceiptRepo
	deletions *read_repo.MessageDeletionRepo
}

type roomProjectionRow = read_repo.RoomProjectionRow
type roomMemberProjectionRow = read_repo.RoomMemberProjectionRow
type messageProjectionRow = read_repo.MessageProjectionRow

func NewCassandraProjectionStore(cfg config.CassandraConfig, session *gocql.Session) (*cassandraProjectionStore, error) {
	if !cfg.Enabled || session == nil {
		return nil, stackErr.Error(fmt.Errorf("cassandra projection store requires an enabled cassandra session"))
	}

	tables := views.DefaultProjectionTableNames()
	store := &cassandraProjectionStore{
		session: session,
		tables:  tables,
	}
	store.rooms = read_repo.NewRoomProjectionRepo(store.session, tables)
	store.messages = read_repo.NewMessageProjectionRepo(store.session, tables)
	store.receipts = read_repo.NewMessageReceiptRepo(store.session, tables)
	store.deletions = read_repo.NewMessageDeletionRepo(store.session, tables)

	if err := runProjectionMigrations(context.Background(), store.session, store.tables); err != nil {
		return nil, stackErr.Error(err)
	}

	return store, nil
}

func (s *cassandraProjectionStore) SyncRoomAggregate(ctx context.Context, projection *roomprojection.RoomAggregateSync) error {
	if s == nil || s.session == nil || projection == nil || projection.Room == nil {
		return nil
	}

	roomID := strings.TrimSpace(projection.Room.RoomID)
	previousRoom, err := s.rooms.GetRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	previousMembers, err := s.rooms.ListRoomMemberRows(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}

	nextRoom := roomProjectionToRow(projection.Room)
	if err := s.rooms.UpsertRoomRow(ctx, nextRoom); err != nil {
		return stackErr.Error(err)
	}

	currentMembers := make(map[string]*roomMemberProjectionRow, len(projection.Members))
	for idx := range projection.Members {
		member := projection.Members[idx]
		memberRow := roomMemberProjectionToRow(&member)
		if memberRow == nil {
			continue
		}
		currentMembers[strings.TrimSpace(memberRow.AccountID)] = memberRow
		if err := s.rooms.UpsertRoomMemberRow(ctx, roomMemberProjectionToRow(&member)); err != nil {
			return stackErr.Error(err)
		}
	}

	for _, member := range previousMembers {
		if member == nil {
			continue
		}
		accountID := strings.TrimSpace(member.AccountID)
		if previousRoom != nil {
			if err := s.rooms.DeleteAccountRoomIndex(ctx, accountID, previousRoom); err != nil {
				return stackErr.Error(err)
			}
		}
		if _, exists := currentMembers[accountID]; !exists {
			if err := s.rooms.DeleteRoomMemberRow(ctx, roomID, accountID); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	for accountID := range currentMembers {
		if err := s.rooms.UpsertAccountRoomIndex(ctx, accountID, nextRoom); err != nil {
			return stackErr.Error(err)
		}
	}

	return nil
}

func (s *cassandraProjectionStore) DeleteRoomAggregate(ctx context.Context, roomID string) error {
	if s == nil || s.session == nil {
		return nil
	}

	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return nil
	}

	roomRow, err := s.rooms.GetRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	members, err := s.rooms.ListRoomMemberRows(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}

	for _, member := range members {
		if member == nil {
			continue
		}
		accountID := strings.TrimSpace(member.AccountID)
		if roomRow != nil {
			if err := s.rooms.DeleteAccountRoomIndex(ctx, accountID, roomRow); err != nil {
				return stackErr.Error(err)
			}
		}
		if err := s.rooms.DeleteRoomMemberRow(ctx, roomID, accountID); err != nil {
			return stackErr.Error(err)
		}
		if err := s.deletions.DeletePartition(ctx, accountID, roomID); err != nil {
			return stackErr.Error(err)
		}
	}

	if err := s.messages.DeleteRoomTimelinePartition(ctx, roomID); err != nil {
		return stackErr.Error(err)
	}
	if err := s.rooms.DeleteRoomRow(ctx, roomID); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (s *cassandraProjectionStore) SyncMessageAggregate(ctx context.Context, projection *roomprojection.MessageAggregateSync) error {
	if s == nil || s.session == nil || projection == nil {
		return nil
	}

	if projection.Message != nil {
		if err := s.messages.UpsertTimelineRow(ctx, projection.Message); err != nil {
			return stackErr.Error(err)
		}
		if err := s.messages.UpsertByIDRow(ctx, projection.Message); err != nil {
			return stackErr.Error(err)
		}
	}

	var roomRow *roomProjectionRow
	roomID := projectionRoomID(projection)
	if len(projection.Members) > 0 && roomID != "" {
		var err error
		roomRow, err = s.rooms.GetRoomRow(ctx, roomID)
		if err != nil {
			return stackErr.Error(err)
		}
	}

	for idx := range projection.Members {
		member := projection.Members[idx]
		if err := s.rooms.UpsertRoomMemberRow(ctx, roomMemberProjectionToRow(&member)); err != nil {
			return stackErr.Error(err)
		}
		if roomRow != nil {
			if err := s.rooms.UpsertAccountRoomIndex(ctx, strings.TrimSpace(member.AccountID), roomRow); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	for idx := range projection.Receipts {
		if err := s.receipts.Upsert(ctx, &projection.Receipts[idx]); err != nil {
			return stackErr.Error(err)
		}
	}
	for idx := range projection.Deletions {
		if err := s.deletions.Upsert(ctx, &projection.Deletions[idx]); err != nil {
			return stackErr.Error(err)
		}
	}

	return nil
}

func (s *cassandraProjectionStore) UpsertRoom(ctx context.Context, room *views.RoomView) error {
	if room == nil {
		return nil
	}
	return stackErr.Error(s.SyncRoomAggregate(ctx, &roomprojection.RoomAggregateSync{
		Room: &roomprojection.RoomProjection{
			RoomID:          room.ID,
			Name:            room.Name,
			Description:     room.Description,
			RoomType:        string(room.RoomType),
			OwnerID:         room.OwnerID,
			PinnedMessageID: utils.DerefString(room.PinnedMessageID),
			MemberCount:     room.MemberCount,
			LastMessage:     roomLastMessageFromView(room),
			CreatedAt:       room.CreatedAt,
			UpdatedAt:       room.UpdatedAt,
		},
	}))
}

func (s *cassandraProjectionStore) UpdateRoom(ctx context.Context, room *views.RoomView) error {
	return stackErr.Error(s.UpsertRoom(ctx, room))
}

func (s *cassandraProjectionStore) DeleteRoom(ctx context.Context, id string) error {
	return stackErr.Error(s.DeleteRoomAggregate(ctx, id))
}

func (s *cassandraProjectionStore) ListRooms(ctx context.Context, options utils.QueryOptions) ([]*views.RoomView, error) {
	return s.listRoomsFromBaseProjection(ctx, options)
}

func (s *cassandraProjectionStore) ListRoomsByAccount(ctx context.Context, accountID string, options utils.QueryOptions) ([]*views.RoomView, error) {
	limit, offset := normalizeOffsetLimit(options.Limit, options.Offset, 20, 100)
	rows, err := s.rooms.ListRoomsByAccount(ctx, accountID, limit, offset)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	results := make([]*views.RoomView, 0, len(rows))
	for _, row := range rows {
		results = append(results, read_repo.RoomRowToEntity(row))
	}
	return results, nil
}

func (s *cassandraProjectionStore) GetRoomByID(ctx context.Context, id string) (*views.RoomView, error) {
	row, err := s.rooms.GetRoomRow(ctx, id)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if row == nil {
		return nil, stackErr.Error(gocql.ErrNotFound)
	}
	return read_repo.RoomRowToEntity(row), nil
}

func (s *cassandraProjectionStore) UpdateRoomStats(ctx context.Context, roomID string, memberCount int, lastMessage *views.MessageView, _ time.Time) error {
	row, err := s.rooms.GetRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if row == nil {
		return nil
	}

	next := cloneRoomRow(row)
	next.MemberCount = memberCount
	if lastMessage == nil {
		next.LastMessageID = ""
		next.LastMessageAt = nil
		next.LastMessageContent = ""
		next.LastMessageSenderID = ""
	} else {
		lastMessageAt := lastMessage.CreatedAt.UTC()
		next.LastMessageID = lastMessage.ID
		next.LastMessageAt = &lastMessageAt
		next.LastMessageContent = lastMessage.Message
		next.LastMessageSenderID = lastMessage.SenderID
	}

	if err := s.rooms.UpsertRoomRow(ctx, next); err != nil {
		return stackErr.Error(err)
	}
	return stackErr.Error(s.syncRoomIndexes(ctx, row, next))
}

func (s *cassandraProjectionStore) UpdatePinnedMessage(ctx context.Context, roomID, pinnedMessageID string, _ time.Time) error {
	row, err := s.rooms.GetRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if row == nil {
		return nil
	}

	next := cloneRoomRow(row)
	next.PinnedMessageID = strings.TrimSpace(pinnedMessageID)

	if err := s.rooms.UpsertRoomRow(ctx, next); err != nil {
		return stackErr.Error(err)
	}
	return stackErr.Error(s.syncRoomIndexes(ctx, row, next))
}

func (s *cassandraProjectionStore) UpsertMessage(ctx context.Context, message *views.MessageView) error {
	if message == nil {
		return nil
	}
	return stackErr.Error(s.SyncMessageAggregate(ctx, &roomprojection.MessageAggregateSync{
		Message: messageViewToProjection(message),
	}))
}

func (s *cassandraProjectionStore) GetMessageByID(ctx context.Context, id string) (*views.MessageView, error) {
	row, err := s.messages.GetMessageByIDRow(ctx, id)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if row == nil {
		return nil, nil
	}

	entityMessage, err := messageRowToEntity(row)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return entityMessage, nil
}

func (s *cassandraProjectionStore) GetLastMessage(ctx context.Context, roomID string) (*views.MessageView, error) {
	row, err := s.messages.GetLastMessageRow(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if row == nil {
		return nil, nil
	}

	entityMessage, err := messageRowToEntity(row)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return entityMessage, nil
}

func (s *cassandraProjectionStore) ListMessages(ctx context.Context, accountID, roomID string, options roomprojection.MessageListOptions) ([]*views.MessageView, error) {
	roomID = strings.TrimSpace(roomID)
	accountID = strings.TrimSpace(accountID)
	if roomID == "" {
		return []*views.MessageView{}, nil
	}

	limit := boundedLimit(options.Limit, 50, 200)
	beforeAt := utils.ClonePtr(options.BeforeAt)

	pageSize := boundedLimit(limit*defaultRoomListPageExpansionFactor, 100, 400)
	collected := make([]*views.MessageView, 0, limit)
	cursor := beforeAt

	for len(collected) < limit {
		batch, err := s.messages.ListTimelineBatch(ctx, roomID, cursor, pageSize, options.Ascending)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		if len(batch) == 0 {
			break
		}

		deletedIDs, err := s.deletions.ListDeletedMessageIDs(ctx, accountID, roomID, batchTimeLowerBound(batch), batchTimeUpperBound(batch))
		if err != nil {
			return nil, stackErr.Error(err)
		}

		for _, row := range batch {
			if _, deleted := deletedIDs[row.MessageID]; deleted {
				continue
			}
			entityMessage, rowErr := messageRowToEntity(row)
			if rowErr != nil {
				return nil, stackErr.Error(rowErr)
			}
			collected = append(collected, entityMessage)
			if len(collected) >= limit {
				break
			}
		}

		if len(batch) < pageSize {
			break
		}
		last := batch[len(batch)-1]
		nextCursor := last.MessageSentAt.UTC()
		cursor = &nextCursor
	}

	if !options.Ascending {
		sort.Slice(collected, func(i, j int) bool {
			return collected[i].CreatedAt.Before(collected[j].CreatedAt)
		})
	}

	return collected, nil
}

func (s *cassandraProjectionStore) UpsertMessageReceipt(
	ctx context.Context,
	messageID,
	accountID,
	status string,
	deliveredAt,
	seenAt *time.Time,
	createdAt,
	updatedAt time.Time,
) error {
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return stackErr.Error(err)
	}
	if message == nil {
		return nil
	}
	return stackErr.Error(s.receipts.Upsert(ctx, &roomprojection.MessageReceiptProjection{
		RoomID:      message.RoomID,
		MessageID:   messageID,
		AccountID:   accountID,
		Status:      status,
		DeliveredAt: deliveredAt,
		SeenAt:      seenAt,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}))
}

func (s *cassandraProjectionStore) GetMessageReceipt(ctx context.Context, lookup roomprojection.MessageReceiptLookup) (*roomprojection.MessageReceiptStatus, error) {
	return s.receipts.GetMessageReceipt(ctx, lookup)
}

func (s *cassandraProjectionStore) CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error) {
	return s.receipts.CountByStatus(ctx, messageID, status)
}

func (s *cassandraProjectionStore) UpsertMessageDeletion(ctx context.Context, messageID, accountID string, createdAt time.Time) error {
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return stackErr.Error(err)
	}
	if message == nil {
		return nil
	}
	return stackErr.Error(s.deletions.Upsert(ctx, &roomprojection.MessageDeletionProjection{
		RoomID:        message.RoomID,
		MessageID:     messageID,
		AccountID:     accountID,
		MessageSentAt: message.CreatedAt,
		CreatedAt:     createdAt,
	}))
}

func (s *cassandraProjectionStore) CountUnreadMessages(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error) {
	roomID = strings.TrimSpace(roomID)
	accountID = strings.TrimSpace(accountID)
	if roomID == "" {
		return 0, nil
	}

	pageSize := 500
	var (
		count  int64
		cursor *time.Time
	)

	if lastReadAt != nil {
		value := lastReadAt.UTC()
		cursor = &value
	}

	for {
		batch, err := s.messages.ListUnreadTimelineBatch(ctx, roomID, cursor, pageSize)
		if err != nil {
			return 0, stackErr.Error(err)
		}
		if len(batch) == 0 {
			break
		}

		deletedIDs, err := s.deletions.ListDeletedMessageIDs(ctx, accountID, roomID, batchTimeLowerBound(batch), batchTimeUpperBound(batch))
		if err != nil {
			return 0, stackErr.Error(err)
		}

		for _, row := range batch {
			if strings.TrimSpace(row.MessageSenderID) == accountID {
				continue
			}
			if row.DeletedForEveryoneAt != nil {
				continue
			}
			if _, deleted := deletedIDs[row.MessageID]; deleted {
				continue
			}
			count++
		}

		if len(batch) < pageSize {
			break
		}
		last := batch[len(batch)-1]
		nextCursor := last.MessageSentAt.UTC()
		cursor = &nextCursor
	}

	return count, nil
}

func (s *cassandraProjectionStore) UpsertRoomMember(ctx context.Context, roomMember *views.RoomMemberView) error {
	if roomMember == nil {
		return nil
	}
	return stackErr.Error(s.SyncMessageAggregate(ctx, &roomprojection.MessageAggregateSync{
		Members: []roomprojection.RoomMemberProjection{
			{
				RoomID:          roomMember.RoomID,
				MemberID:        roomMember.ID,
				AccountID:       roomMember.AccountID,
				DisplayName:     roomMember.DisplayName,
				Username:        roomMember.Username,
				AvatarObjectKey: roomMember.AvatarObjectKey,
				Role:            string(roomMember.Role),
				LastDeliveredAt: utils.ClonePtr(roomMember.LastDeliveredAt),
				LastReadAt:      utils.ClonePtr(roomMember.LastReadAt),
				CreatedAt:       roomMember.CreatedAt,
				UpdatedAt:       roomMember.UpdatedAt,
			},
		},
	}))
}

func (s *cassandraProjectionStore) DeleteRoomMember(ctx context.Context, roomID, accountID string) error {
	if err := s.rooms.DeleteRoomMemberRow(ctx, roomID, accountID); err != nil {
		return stackErr.Error(err)
	}

	roomRow, err := s.rooms.GetRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if roomRow == nil {
		return nil
	}
	return stackErr.Error(s.rooms.DeleteAccountRoomIndex(ctx, strings.TrimSpace(accountID), roomRow))
}

func (s *cassandraProjectionStore) ListRoomMembers(ctx context.Context, roomID string) ([]*views.RoomMemberView, error) {
	rows, err := s.rooms.ListRoomMemberRows(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	results := make([]*views.RoomMemberView, 0, len(rows))
	for _, row := range rows {
		results = append(results, read_repo.RoomMemberRowToEntity(row))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})
	return results, nil
}

func (s *cassandraProjectionStore) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error) {
	row, err := s.rooms.GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return read_repo.RoomMemberRowToEntity(row), nil
}

func (s *cassandraProjectionStore) SearchMentionCandidates(ctx context.Context, search roomprojection.MentionCandidateSearch) ([]*views.MentionCandidateView, error) {
	return nil, nil
}

func (s *cassandraProjectionStore) syncRoomIndexes(ctx context.Context, previous, current *roomProjectionRow) error {
	if current == nil {
		return nil
	}

	members, err := s.rooms.ListRoomMemberRows(ctx, current.RoomID)
	if err != nil {
		return stackErr.Error(err)
	}

	if previous != nil && !previous.UpdatedAt.Equal(current.UpdatedAt) {
		for _, member := range members {
			if err := s.rooms.DeleteAccountRoomIndex(ctx, strings.TrimSpace(member.AccountID), previous); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	for _, member := range members {
		if err := s.rooms.UpsertAccountRoomIndex(ctx, strings.TrimSpace(member.AccountID), current); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func messageRowToEntity(row *messageProjectionRow) (*views.MessageView, error) {
	if row == nil {
		return nil, nil
	}

	mentions, err := unmarshalProjectionMentions(row.MentionsJSON)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	reactions, err := unmarshalProjectionReactions(row.ReactionsJSON)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &views.MessageView{
		ID:                     row.MessageID,
		RoomID:                 row.RoomID,
		SenderID:               row.MessageSenderID,
		Message:                row.MessageContent,
		MessageType:            row.MessageType,
		Mentions:               mentions,
		Reactions:              reactions,
		MentionAll:             row.MentionAll,
		ReplyToMessageID:       strings.TrimSpace(row.ReplyToMessageID),
		ForwardedFromMessageID: strings.TrimSpace(row.ForwardedFromMessageID),
		FileName:               strings.TrimSpace(row.FileName),
		FileSize:               row.FileSize,
		MimeType:               strings.TrimSpace(row.MimeType),
		ObjectKey:              strings.TrimSpace(row.ObjectKey),
		EditedAt:               utils.ClonePtr(row.EditedAt),
		DeletedForEveryoneAt:   utils.ClonePtr(row.DeletedForEveryoneAt),
		CreatedAt:              row.MessageSentAt.UTC(),
	}, nil
}

func roomProjectionToRow(projection *roomprojection.RoomProjection) *roomProjectionRow {
	if projection == nil {
		return nil
	}

	row := &roomProjectionRow{
		RoomID:          strings.TrimSpace(projection.RoomID),
		Name:            projection.Name,
		Description:     projection.Description,
		RoomType:        projection.RoomType,
		OwnerID:         projection.OwnerID,
		PinnedMessageID: strings.TrimSpace(projection.PinnedMessageID),
		MemberCount:     projection.MemberCount,
		CreatedAt:       projection.CreatedAt.UTC(),
		UpdatedAt:       projection.UpdatedAt.UTC(),
	}
	if projection.LastMessage != nil {
		row.LastMessageID = strings.TrimSpace(projection.LastMessage.MessageID)
		row.LastMessageAt = utils.ClonePtr(projection.LastMessage.MessageSentAt)
		row.LastMessageContent = projection.LastMessage.MessageContent
		row.LastMessageSenderID = projection.LastMessage.MessageSenderID
	}
	return row
}

func roomMemberProjectionToRow(projection *roomprojection.RoomMemberProjection) *roomMemberProjectionRow {
	if projection == nil {
		return nil
	}

	return &roomMemberProjectionRow{
		RoomID:          strings.TrimSpace(projection.RoomID),
		MemberID:        strings.TrimSpace(projection.MemberID),
		AccountID:       strings.TrimSpace(projection.AccountID),
		DisplayName:     strings.TrimSpace(projection.DisplayName),
		Username:        strings.TrimSpace(projection.Username),
		AvatarObjectKey: strings.TrimSpace(projection.AvatarObjectKey),
		Role:            projection.Role,
		LastDeliveredAt: utils.ClonePtr(projection.LastDeliveredAt),
		LastReadAt:      utils.ClonePtr(projection.LastReadAt),
		CreatedAt:       projection.CreatedAt.UTC(),
		UpdatedAt:       projection.UpdatedAt.UTC(),
	}
}

func messageViewToProjection(message *views.MessageView) *roomprojection.MessageProjection {
	if message == nil {
		return nil
	}

	mentions := make([]roomprojection.ProjectionMention, 0, len(message.Mentions))
	mentionedAccountIDs := make([]string, 0, len(message.Mentions))
	for _, mention := range message.Mentions {
		mentions = append(mentions, roomprojection.ProjectionMention{
			AccountID:   strings.TrimSpace(mention.AccountID),
			DisplayName: strings.TrimSpace(mention.DisplayName),
			Username:    strings.TrimSpace(mention.Username),
		})
		mentionedAccountIDs = append(mentionedAccountIDs, strings.TrimSpace(mention.AccountID))
	}

	return &roomprojection.MessageProjection{
		RoomID:                 message.RoomID,
		MessageID:              message.ID,
		MessageContent:         message.Message,
		MessageType:            message.MessageType,
		ReplyToMessageID:       message.ReplyToMessageID,
		ForwardedFromMessageID: message.ForwardedFromMessageID,
		FileName:               message.FileName,
		FileSize:               message.FileSize,
		MimeType:               message.MimeType,
		ObjectKey:              message.ObjectKey,
		MessageSenderID:        message.SenderID,
		MessageSentAt:          message.CreatedAt.UTC(),
		Mentions:               mentions,
		Reactions:              mapProjectionReactionsFromView(message.Reactions),
		MentionAll:             message.MentionAll,
		MentionedAccountIDs:    mentionedAccountIDs,
		EditedAt:               utils.ClonePtr(message.EditedAt),
		DeletedForEveryoneAt:   utils.ClonePtr(message.DeletedForEveryoneAt),
	}
}

func roomLastMessageFromView(room *views.RoomView) *roomprojection.RoomLastMessageProjection {
	if room == nil || room.LastMessageID == nil {
		return nil
	}
	return &roomprojection.RoomLastMessageProjection{
		MessageID:       strings.TrimSpace(*room.LastMessageID),
		MessageSentAt:   utils.ClonePtr(room.LastMessageAt),
		MessageContent:  utils.DerefString(room.LastMessageContent),
		MessageSenderID: utils.DerefString(room.LastMessageSenderID),
	}
}

func projectionRoomID(projection *roomprojection.MessageAggregateSync) string {
	if projection == nil {
		return ""
	}
	if projection.Message != nil {
		return strings.TrimSpace(projection.Message.RoomID)
	}
	if len(projection.Members) > 0 {
		return strings.TrimSpace(projection.Members[0].RoomID)
	}
	if len(projection.Receipts) > 0 {
		return strings.TrimSpace(projection.Receipts[0].RoomID)
	}
	if len(projection.Deletions) > 0 {
		return strings.TrimSpace(projection.Deletions[0].RoomID)
	}
	return ""
}

func cloneRoomRow(row *roomProjectionRow) *roomProjectionRow {
	if row == nil {
		return nil
	}
	copy := *row
	copy.LastMessageAt = utils.ClonePtr(row.LastMessageAt)
	return &copy
}

func batchTimeLowerBound(rows []*messageProjectionRow) *time.Time {
	if len(rows) == 0 {
		return nil
	}
	min := rows[0].MessageSentAt.UTC()
	for _, row := range rows[1:] {
		if row.MessageSentAt.Before(min) {
			min = row.MessageSentAt.UTC()
		}
	}
	return &min
}

func batchTimeUpperBound(rows []*messageProjectionRow) *time.Time {
	if len(rows) == 0 {
		return nil
	}
	max := rows[0].MessageSentAt.UTC()
	for _, row := range rows[1:] {
		if row.MessageSentAt.After(max) {
			max = row.MessageSentAt.UTC()
		}
	}
	return &max
}

func (s *cassandraProjectionStore) listRoomsFromBaseProjection(ctx context.Context, options utils.QueryOptions) ([]*views.RoomView, error) {
	limit, offset := normalizeOffsetLimit(options.Limit, options.Offset, 20, 100)
	rows, err := s.rooms.ListRoomsFromBaseProjection(ctx, limit, offset)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	results := make([]*views.RoomView, 0, len(rows))
	for _, row := range rows {
		results = append(results, read_repo.RoomRowToEntity(row))
	}
	return results, nil
}

func normalizeOffsetLimit(limit *int, offset *int, defaultLimit, maxLimit int) (int, int) {
	valueLimit := defaultLimit
	if limit != nil && *limit > 0 {
		valueLimit = *limit
	}
	if valueLimit > maxLimit {
		valueLimit = maxLimit
	}

	valueOffset := 0
	if offset != nil && *offset > 0 {
		valueOffset = *offset
	}
	return valueLimit, valueOffset
}

func boundedLimit(value, defaultValue, maxValue int) int {
	if value <= 0 {
		value = defaultValue
	}
	if value > maxValue {
		value = maxValue
	}
	return value
}

func unmarshalProjectionMentions(raw string) ([]views.MessageMentionView, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var items []roomprojection.ProjectionMention
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, stackErr.Error(err)
	}
	results := make([]views.MessageMentionView, 0, len(items))
	for _, item := range items {
		results = append(results, views.MessageMentionView{
			AccountID:   item.AccountID,
			DisplayName: item.DisplayName,
			Username:    item.Username,
		})
	}
	return results, nil
}

func unmarshalProjectionReactions(raw string) ([]views.MessageReactionView, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var items []roomprojection.ProjectionReaction
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, stackErr.Error(err)
	}

	results := make([]views.MessageReactionView, 0, len(items))
	for _, item := range items {
		results = append(results, views.MessageReactionView{
			AccountID: strings.TrimSpace(item.AccountID),
			Emoji:     strings.TrimSpace(item.Emoji),
			ReactedAt: item.ReactedAt.UTC(),
		})
	}
	return results, nil
}

func mapProjectionReactionsFromView(items []views.MessageReactionView) []roomprojection.ProjectionReaction {
	if len(items) == 0 {
		return nil
	}

	results := make([]roomprojection.ProjectionReaction, 0, len(items))
	for _, item := range items {
		results = append(results, roomprojection.ProjectionReaction{
			AccountID: strings.TrimSpace(item.AccountID),
			Emoji:     strings.TrimSpace(item.Emoji),
			ReactedAt: item.ReactedAt.UTC(),
		})
	}
	return results
}
