package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	roomprojection "go-socket/core/modules/room/application/projection"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"

	"github.com/gocql/gocql"
)

const (
	defaultRoomProjectionTable         = "room_projections_by_id"
	defaultRoomProjectionByAccount     = "room_projections_by_account"
	defaultRoomProjectionGlobal        = "room_projections_global"
	defaultRoomMemberProjectionTable   = "room_member_projections_by_room"
	defaultRoomMessageByIDTable        = "room_messages_by_id"
	defaultRoomMessageReceiptTable     = "room_message_receipts_by_message"
	defaultRoomMessageDeletionTable    = "room_message_deletions_by_account_room"
	globalRoomProjectionPartition      = "all"
	defaultRoomListPageExpansionFactor = 3
)

type cassandraProjectionStore struct {
	session              *gocql.Session
	roomTable            string
	roomsByAccountTable  string
	roomMembersTable     string
	roomTimelineTable    string
	messageByIDTable     string
	messageReceiptsTable string
	messageDeletesTable  string
}

type roomProjectionRow struct {
	RoomID              string
	Name                string
	Description         string
	RoomType            string
	OwnerID             string
	PinnedMessageID     string
	MemberCount         int
	LastMessageID       string
	LastMessageAt       *time.Time
	LastMessageContent  string
	LastMessageSenderID string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type roomMemberProjectionRow struct {
	RoomID          string
	MemberID        string
	AccountID       string
	DisplayName     string
	Username        string
	AvatarObjectKey string
	Role            string
	LastDeliveredAt *time.Time
	LastReadAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type messageProjectionRow struct {
	RoomID                 string
	RoomName               string
	RoomType               string
	MessageID              string
	MessageContent         string
	MessageType            string
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
	MessageSenderID        string
	MessageSenderName      string
	MessageSenderEmail     string
	MessageSentAt          time.Time
	MentionsJSON           string
	MentionAll             bool
	MentionedAccountIDs    []string
	EditedAt               *time.Time
	DeletedForEveryoneAt   *time.Time
}

func NewCassandraProjectionStore(cfg config.CassandraConfig, session *gocql.Session) (*cassandraProjectionStore, error) {
	if !cfg.Enabled || session == nil {
		return nil, stackErr.Error(fmt.Errorf("cassandra projection store requires an enabled cassandra session"))
	}

	store := &cassandraProjectionStore{
		session:              session,
		roomTable:            normalizeProjectionTable(defaultRoomProjectionTable),
		roomsByAccountTable:  normalizeProjectionTable(defaultRoomProjectionByAccount),
		roomMembersTable:     normalizeProjectionTable(defaultRoomMemberProjectionTable),
		roomTimelineTable:    normalizeTimelineTable(cfg.RoomTimelineTable),
		messageByIDTable:     normalizeProjectionTable(defaultRoomMessageByIDTable),
		messageReceiptsTable: normalizeProjectionTable(defaultRoomMessageReceiptTable),
		messageDeletesTable:  normalizeProjectionTable(defaultRoomMessageDeletionTable),
	}

	if err := store.ensureSchema(context.Background()); err != nil {
		return nil, stackErr.Error(err)
	}

	return store, nil
}

func (s *cassandraProjectionStore) ensureSchema(ctx context.Context) error {
	statements := []string{
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				room_id text PRIMARY KEY,
				name text,
				description text,
				room_type text,
				owner_id text,
				pinned_message_id text,
				member_count int,
				last_message_id text,
				last_message_at timestamp,
				last_message_content text,
				last_message_sender_id text,
				created_at timestamp,
				updated_at timestamp
			)
		`, s.roomTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				account_id text,
				room_updated_at timestamp,
				room_id text,
				name text,
				description text,
				room_type text,
				owner_id text,
				pinned_message_id text,
				member_count int,
				last_message_id text,
				last_message_at timestamp,
				last_message_content text,
				last_message_sender_id text,
				created_at timestamp,
				PRIMARY KEY ((account_id), room_updated_at, room_id)
			) WITH CLUSTERING ORDER BY (room_updated_at DESC, room_id DESC)
		`, s.roomsByAccountTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				room_id text,
				account_id text,
				member_id text,
				display_name text,
				username text,
				avatar_object_key text,
				role text,
				last_delivered_at timestamp,
				last_read_at timestamp,
				created_at timestamp,
				updated_at timestamp,
				PRIMARY KEY ((room_id), account_id)
			)
		`, s.roomMembersTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				room_id text,
				message_sent_at timestamp,
				message_id text,
				room_name text,
				room_type text,
				message_content text,
				message_type text,
				reply_to_message_id text,
				forwarded_from_message_id text,
				file_name text,
				file_size bigint,
				mime_type text,
				object_key text,
				message_sender_id text,
				message_sender_name text,
				message_sender_email text,
				mentions_json text,
				mention_all boolean,
				mentioned_account_ids list<text>,
				edited_at timestamp,
				deleted_for_everyone_at timestamp,
				PRIMARY KEY ((room_id), message_sent_at, message_id)
			) WITH CLUSTERING ORDER BY (message_sent_at DESC, message_id DESC)
		`, s.roomTimelineTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				message_id text PRIMARY KEY,
				room_id text,
				room_name text,
				room_type text,
				message_content text,
				message_type text,
				reply_to_message_id text,
				forwarded_from_message_id text,
				file_name text,
				file_size bigint,
				mime_type text,
				object_key text,
				message_sender_id text,
				message_sender_name text,
				message_sender_email text,
				message_sent_at timestamp,
				mentions_json text,
				mention_all boolean,
				mentioned_account_ids list<text>,
				edited_at timestamp,
				deleted_for_everyone_at timestamp
			)
		`, s.messageByIDTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				message_id text,
				account_id text,
				room_id text,
				status text,
				delivered_at timestamp,
				seen_at timestamp,
				created_at timestamp,
				updated_at timestamp,
				PRIMARY KEY ((message_id), account_id)
			)
		`, s.messageReceiptsTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				account_id text,
				room_id text,
				message_sent_at timestamp,
				message_id text,
				created_at timestamp,
				PRIMARY KEY ((account_id, room_id), message_sent_at, message_id)
			) WITH CLUSTERING ORDER BY (message_sent_at DESC, message_id DESC)
		`, s.messageDeletesTable),
	}

	for _, statement := range statements {
		if err := s.session.Query(statement).WithContext(ctx).Exec(); err != nil {
			return stackErr.Error(fmt.Errorf("ensure cassandra room projection schema failed: %v", err))
		}
	}

	if err := s.ensureRoomMemberProjectionColumns(ctx); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (s *cassandraProjectionStore) ensureRoomMemberProjectionColumns(ctx context.Context) error {
	statements := []string{
		fmt.Sprintf(`ALTER TABLE %s ADD display_name text`, s.roomMembersTable),
		fmt.Sprintf(`ALTER TABLE %s ADD username text`, s.roomMembersTable),
		fmt.Sprintf(`ALTER TABLE %s ADD avatar_object_key text`, s.roomMembersTable),
	}

	for _, statement := range statements {
		if err := s.session.Query(statement).WithContext(ctx).Exec(); err != nil {
			if isCassandraExistingColumnError(err) {
				continue
			}
			return stackErr.Error(fmt.Errorf("ensure cassandra room member projection columns failed: %v", err))
		}
	}

	return nil
}

func isCassandraExistingColumnError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

func (s *cassandraProjectionStore) SyncRoomAggregate(ctx context.Context, projection *roomprojection.RoomAggregateSync) error {
	if s == nil || s.session == nil || projection == nil || projection.Room == nil {
		return nil
	}

	roomID := strings.TrimSpace(projection.Room.RoomID)
	previousRoom, err := s.getRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	previousMembers, err := s.listRoomMemberRows(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}

	nextRoom := roomProjectionToRow(projection.Room)
	if err := s.upsertRoomRow(ctx, nextRoom); err != nil {
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
		if err := s.upsertRoomMemberRow(ctx, &member); err != nil {
			return stackErr.Error(err)
		}
	}

	for _, member := range previousMembers {
		if member == nil {
			continue
		}
		accountID := strings.TrimSpace(member.AccountID)
		if previousRoom != nil {
			if err := s.deleteAccountRoomIndex(ctx, accountID, previousRoom); err != nil {
				return stackErr.Error(err)
			}
		}
		if _, exists := currentMembers[accountID]; !exists {
			if err := s.deleteRoomMemberRow(ctx, roomID, accountID); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	for accountID := range currentMembers {
		if err := s.upsertAccountRoomIndex(ctx, accountID, nextRoom); err != nil {
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

	roomRow, err := s.getRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	members, err := s.listRoomMemberRows(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}

	for _, member := range members {
		if member == nil {
			continue
		}
		accountID := strings.TrimSpace(member.AccountID)
		if roomRow != nil {
			if err := s.deleteAccountRoomIndex(ctx, accountID, roomRow); err != nil {
				return stackErr.Error(err)
			}
		}
		if err := s.deleteRoomMemberRow(ctx, roomID, accountID); err != nil {
			return stackErr.Error(err)
		}
		if err := s.deleteMessageDeletePartition(ctx, accountID, roomID); err != nil {
			return stackErr.Error(err)
		}
	}

	if err := s.deleteRoomTimelinePartition(ctx, roomID); err != nil {
		return stackErr.Error(err)
	}
	if err := s.deleteRoomRow(ctx, roomID); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (s *cassandraProjectionStore) SyncMessageAggregate(ctx context.Context, projection *roomprojection.MessageAggregateSync) error {
	if s == nil || s.session == nil || projection == nil {
		return nil
	}

	if projection.Message != nil {
		if err := s.upsertTimelineMessageRow(ctx, projection.Message); err != nil {
			return stackErr.Error(err)
		}
		if err := s.upsertMessageByIDRow(ctx, projection.Message); err != nil {
			return stackErr.Error(err)
		}
	}

	var roomRow *roomProjectionRow
	roomID := projectionRoomID(projection)
	if len(projection.Members) > 0 && roomID != "" {
		var err error
		roomRow, err = s.getRoomRow(ctx, roomID)
		if err != nil {
			return stackErr.Error(err)
		}
	}

	for idx := range projection.Members {
		member := projection.Members[idx]
		if err := s.upsertRoomMemberRow(ctx, &member); err != nil {
			return stackErr.Error(err)
		}
		if roomRow != nil {
			if err := s.upsertAccountRoomIndex(ctx, strings.TrimSpace(member.AccountID), roomRow); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	for idx := range projection.Receipts {
		if err := s.upsertMessageReceiptRow(ctx, &projection.Receipts[idx]); err != nil {
			return stackErr.Error(err)
		}
	}
	for idx := range projection.Deletions {
		if err := s.upsertMessageDeletionRow(ctx, &projection.Deletions[idx]); err != nil {
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
			PinnedMessageID: derefProjectionString(room.PinnedMessageID),
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
	queryLimit := limitWithOffset(limit, offset)

	statement := fmt.Sprintf(`
		SELECT
			room_id,
			name,
			description,
			room_type,
			owner_id,
			pinned_message_id,
			member_count,
			last_message_id,
			last_message_at,
			last_message_content,
			last_message_sender_id,
			created_at,
			room_updated_at
		FROM %s
		WHERE account_id = ?
		LIMIT ?
	`, s.roomsByAccountTable)

	rows := make([]*roomProjectionRow, 0, queryLimit)
	iter := s.session.Query(statement, strings.TrimSpace(accountID), queryLimit).WithContext(ctx).Iter()
	defer iter.Close()

	var (
		roomID              string
		name                string
		description         string
		roomType            string
		ownerID             string
		pinnedMessageID     string
		memberCount         int
		lastMessageID       string
		lastMessageAt       *time.Time
		lastMessageContent  string
		lastMessageSenderID string
		createdAt           time.Time
		updatedAt           time.Time
	)
	scanner := iter.Scanner()
	for scanner.Next() {
		if err := scanner.Scan(
			&roomID,
			&name,
			&description,
			&roomType,
			&ownerID,
			&pinnedMessageID,
			&memberCount,
			&lastMessageID,
			&lastMessageAt,
			&lastMessageContent,
			&lastMessageSenderID,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, stackErr.Error(fmt.Errorf("scan cassandra account room projection failed: %v", err))
		}
		rows = append(rows, &roomProjectionRow{
			RoomID:              roomID,
			Name:                name,
			Description:         description,
			RoomType:            roomType,
			OwnerID:             ownerID,
			PinnedMessageID:     pinnedMessageID,
			MemberCount:         memberCount,
			LastMessageID:       lastMessageID,
			LastMessageAt:       cloneTime(lastMessageAt),
			LastMessageContent:  lastMessageContent,
			LastMessageSenderID: lastMessageSenderID,
			CreatedAt:           createdAt.UTC(),
			UpdatedAt:           updatedAt.UTC(),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate cassandra account room projections failed: %v", err))
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("close cassandra account room projection iterator failed: %v", err))
	}

	return sliceRoomEntities(rows, offset, limit), nil
}

func (s *cassandraProjectionStore) GetRoomByID(ctx context.Context, id string) (*views.RoomView, error) {
	row, err := s.getRoomRow(ctx, id)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if row == nil {
		return nil, stackErr.Error(gocql.ErrNotFound)
	}
	return roomRowToEntity(row), nil
}

func (s *cassandraProjectionStore) UpdateRoomStats(ctx context.Context, roomID string, memberCount int, lastMessage *views.MessageView, _ time.Time) error {
	row, err := s.getRoomRow(ctx, roomID)
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

	if err := s.upsertRoomRow(ctx, next); err != nil {
		return stackErr.Error(err)
	}
	return stackErr.Error(s.syncRoomIndexes(ctx, row, next))
}

func (s *cassandraProjectionStore) UpdatePinnedMessage(ctx context.Context, roomID, pinnedMessageID string, _ time.Time) error {
	row, err := s.getRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if row == nil {
		return nil
	}

	next := cloneRoomRow(row)
	next.PinnedMessageID = strings.TrimSpace(pinnedMessageID)

	if err := s.upsertRoomRow(ctx, next); err != nil {
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
	statement := fmt.Sprintf(`
		SELECT
			room_id,
			room_name,
			room_type,
			message_id,
			message_content,
			message_type,
			reply_to_message_id,
			forwarded_from_message_id,
			file_name,
			file_size,
			mime_type,
			object_key,
			message_sender_id,
			message_sender_name,
			message_sender_email,
			message_sent_at,
			mentions_json,
			mention_all,
			mentioned_account_ids,
			edited_at,
			deleted_for_everyone_at
		FROM %s
		WHERE message_id = ?
	`, s.messageByIDTable)

	row := &messageProjectionRow{}
	if err := s.session.Query(statement, strings.TrimSpace(id)).WithContext(ctx).Scan(
		&row.RoomID,
		&row.RoomName,
		&row.RoomType,
		&row.MessageID,
		&row.MessageContent,
		&row.MessageType,
		&row.ReplyToMessageID,
		&row.ForwardedFromMessageID,
		&row.FileName,
		&row.FileSize,
		&row.MimeType,
		&row.ObjectKey,
		&row.MessageSenderID,
		&row.MessageSenderName,
		&row.MessageSenderEmail,
		&row.MessageSentAt,
		&row.MentionsJSON,
		&row.MentionAll,
		&row.MentionedAccountIDs,
		&row.EditedAt,
		&row.DeletedForEveryoneAt,
	); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	entityMessage, err := messageRowToEntity(row)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return entityMessage, nil
}

func (s *cassandraProjectionStore) GetLastMessage(ctx context.Context, roomID string) (*views.MessageView, error) {
	statement := fmt.Sprintf(`
		SELECT
			room_id,
			room_name,
			room_type,
			message_id,
			message_content,
			message_type,
			reply_to_message_id,
			forwarded_from_message_id,
			file_name,
			file_size,
			mime_type,
			object_key,
			message_sender_id,
			message_sender_name,
			message_sender_email,
			message_sent_at,
			mentions_json,
			mention_all,
			mentioned_account_ids,
			edited_at,
			deleted_for_everyone_at
		FROM %s
		WHERE room_id = ?
		LIMIT 1
	`, s.roomTimelineTable)

	row := &messageProjectionRow{}
	if err := s.session.Query(statement, strings.TrimSpace(roomID)).WithContext(ctx).Scan(
		&row.RoomID,
		&row.RoomName,
		&row.RoomType,
		&row.MessageID,
		&row.MessageContent,
		&row.MessageType,
		&row.ReplyToMessageID,
		&row.ForwardedFromMessageID,
		&row.FileName,
		&row.FileSize,
		&row.MimeType,
		&row.ObjectKey,
		&row.MessageSenderID,
		&row.MessageSenderName,
		&row.MessageSenderEmail,
		&row.MessageSentAt,
		&row.MentionsJSON,
		&row.MentionAll,
		&row.MentionedAccountIDs,
		&row.EditedAt,
		&row.DeletedForEveryoneAt,
	); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
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
	beforeAt := cloneTime(options.BeforeAt)

	pageSize := boundedLimit(limit*defaultRoomListPageExpansionFactor, 100, 400)
	collected := make([]*views.MessageView, 0, limit)
	cursor := beforeAt

	for len(collected) < limit {
		batch, err := s.listTimelineBatch(ctx, roomID, cursor, pageSize, options.Ascending)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		if len(batch) == 0 {
			break
		}

		deletedIDs, err := s.listDeletedMessageIDs(ctx, accountID, roomID, batchTimeLowerBound(batch), batchTimeUpperBound(batch))
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
	return stackErr.Error(s.upsertMessageReceiptRow(ctx, &roomprojection.MessageReceiptProjection{
		RoomID:      roomIDFromMessageView(message),
		MessageID:   messageID,
		AccountID:   accountID,
		Status:      status,
		DeliveredAt: deliveredAt,
		SeenAt:      seenAt,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}))
}

func (s *cassandraProjectionStore) GetMessageReceipt(ctx context.Context, messageID, accountID string) (string, *time.Time, *time.Time, error) {
	statement := fmt.Sprintf(`
		SELECT status, delivered_at, seen_at
		FROM %s
		WHERE message_id = ? AND account_id = ?
	`, s.messageReceiptsTable)

	var (
		status      string
		deliveredAt *time.Time
		seenAt      *time.Time
	)
	if err := s.session.Query(statement, strings.TrimSpace(messageID), strings.TrimSpace(accountID)).WithContext(ctx).Scan(
		&status,
		&deliveredAt,
		&seenAt,
	); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return "", nil, nil, nil
		}
		return "", nil, nil, stackErr.Error(err)
	}
	return status, cloneTime(deliveredAt), cloneTime(seenAt), nil
}

func (s *cassandraProjectionStore) CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error) {
	statement := fmt.Sprintf(`
		SELECT status
		FROM %s
		WHERE message_id = ?
	`, s.messageReceiptsTable)

	iter := s.session.Query(statement, strings.TrimSpace(messageID)).WithContext(ctx).Iter()
	defer iter.Close()

	var (
		value string
		count int64
	)
	scanner := iter.Scanner()
	for scanner.Next() {
		if err := scanner.Scan(&value); err != nil {
			return 0, stackErr.Error(fmt.Errorf("scan cassandra message receipt failed: %v", err))
		}
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(status)) {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, stackErr.Error(fmt.Errorf("iterate cassandra message receipts failed: %v", err))
	}
	if err := iter.Close(); err != nil {
		return 0, stackErr.Error(fmt.Errorf("close cassandra message receipts iterator failed: %v", err))
	}
	return count, nil
}

func (s *cassandraProjectionStore) UpsertMessageDeletion(ctx context.Context, messageID, accountID string, createdAt time.Time) error {
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return stackErr.Error(err)
	}
	if message == nil {
		return nil
	}
	return stackErr.Error(s.upsertMessageDeletionRow(ctx, &roomprojection.MessageDeletionProjection{
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
		batch, err := s.listUnreadTimelineBatch(ctx, roomID, cursor, pageSize)
		if err != nil {
			return 0, stackErr.Error(err)
		}
		if len(batch) == 0 {
			break
		}

		deletedIDs, err := s.listDeletedMessageIDs(ctx, accountID, roomID, batchTimeLowerBound(batch), batchTimeUpperBound(batch))
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
				LastDeliveredAt: cloneTime(roomMember.LastDeliveredAt),
				LastReadAt:      cloneTime(roomMember.LastReadAt),
				CreatedAt:       roomMember.CreatedAt,
				UpdatedAt:       roomMember.UpdatedAt,
			},
		},
	}))
}

func (s *cassandraProjectionStore) DeleteRoomMember(ctx context.Context, roomID, accountID string) error {
	if err := s.deleteRoomMemberRow(ctx, roomID, accountID); err != nil {
		return stackErr.Error(err)
	}

	roomRow, err := s.getRoomRow(ctx, roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if roomRow == nil {
		return nil
	}
	return stackErr.Error(s.deleteAccountRoomIndex(ctx, strings.TrimSpace(accountID), roomRow))
}

func (s *cassandraProjectionStore) ListRoomMembers(ctx context.Context, roomID string) ([]*views.RoomMemberView, error) {
	rows, err := s.listRoomMemberRows(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	results := make([]*views.RoomMemberView, 0, len(rows))
	for _, row := range rows {
		results = append(results, roomMemberRowToEntity(row))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})
	return results, nil
}

func (s *cassandraProjectionStore) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error) {
	statement := fmt.Sprintf(`
		SELECT
			room_id,
			account_id,
			member_id,
			display_name,
			username,
			avatar_object_key,
			role,
			last_delivered_at,
			last_read_at,
			created_at,
			updated_at
		FROM %s
		WHERE room_id = ? AND account_id = ?
	`, s.roomMembersTable)

	row := &roomMemberProjectionRow{}
	if err := s.session.Query(statement, strings.TrimSpace(roomID), strings.TrimSpace(accountID)).WithContext(ctx).Scan(
		&row.RoomID,
		&row.AccountID,
		&row.MemberID,
		&row.DisplayName,
		&row.Username,
		&row.AvatarObjectKey,
		&row.Role,
		&row.LastDeliveredAt,
		&row.LastReadAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	return roomMemberRowToEntity(row), nil
}

func (s *cassandraProjectionStore) SearchMentionCandidates(ctx context.Context, roomID, keyword, excludeAccountID string, limit int) ([]*views.MentionCandidateView, error) {
	return nil, nil
}

func (s *cassandraProjectionStore) listRoomMemberRows(ctx context.Context, roomID string) ([]*roomMemberProjectionRow, error) {
	statement := fmt.Sprintf(`
		SELECT
			room_id,
			account_id,
			member_id,
			display_name,
			username,
			avatar_object_key,
			role,
			last_delivered_at,
			last_read_at,
			created_at,
			updated_at
		FROM %s
		WHERE room_id = ?
	`, s.roomMembersTable)

	rows := make([]*roomMemberProjectionRow, 0)
	iter := s.session.Query(statement, strings.TrimSpace(roomID)).WithContext(ctx).Iter()
	defer iter.Close()

	var (
		lastDeliveredAt *time.Time
		lastReadAt      *time.Time
	)
	scanner := iter.Scanner()
	for scanner.Next() {
		row := &roomMemberProjectionRow{}
		lastDeliveredAt = nil
		lastReadAt = nil
		if err := scanner.Scan(
			&row.RoomID,
			&row.AccountID,
			&row.MemberID,
			&row.DisplayName,
			&row.Username,
			&row.AvatarObjectKey,
			&row.Role,
			&lastDeliveredAt,
			&lastReadAt,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, stackErr.Error(fmt.Errorf("scan cassandra room member projection failed: %v", err))
		}
		row.LastDeliveredAt = cloneTime(lastDeliveredAt)
		row.LastReadAt = cloneTime(lastReadAt)
		row.CreatedAt = row.CreatedAt.UTC()
		row.UpdatedAt = row.UpdatedAt.UTC()
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate cassandra room member projections failed: %v", err))
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("close cassandra room member projection iterator failed: %v", err))
	}
	return rows, nil
}

func (s *cassandraProjectionStore) upsertRoomMemberRow(ctx context.Context, projection *roomprojection.RoomMemberProjection) error {
	statement := fmt.Sprintf(`
		INSERT INTO %s (
			room_id,
			account_id,
			member_id,
			display_name,
			username,
			avatar_object_key,
			role,
			last_delivered_at,
			last_read_at,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.roomMembersTable)

	if err := s.session.Query(
		statement,
		projection.RoomID,
		projection.AccountID,
		projection.MemberID,
		nullableProjectionString(projection.DisplayName),
		nullableProjectionString(projection.Username),
		nullableProjectionString(projection.AvatarObjectKey),
		projection.Role,
		projection.LastDeliveredAt,
		projection.LastReadAt,
		projection.CreatedAt.UTC(),
		projection.UpdatedAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra room member projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) upsertMessageReceiptRow(ctx context.Context, projection *roomprojection.MessageReceiptProjection) error {
	if projection == nil {
		return nil
	}

	statement := fmt.Sprintf(`
		INSERT INTO %s (
			message_id,
			account_id,
			room_id,
			status,
			delivered_at,
			seen_at,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, s.messageReceiptsTable)

	if err := s.session.Query(
		statement,
		projection.MessageID,
		projection.AccountID,
		projection.RoomID,
		projection.Status,
		projection.DeliveredAt,
		projection.SeenAt,
		projection.CreatedAt.UTC(),
		projection.UpdatedAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra message receipt projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) upsertMessageDeletionRow(ctx context.Context, projection *roomprojection.MessageDeletionProjection) error {
	if projection == nil {
		return nil
	}

	statement := fmt.Sprintf(`
		INSERT INTO %s (
			account_id,
			room_id,
			message_sent_at,
			message_id,
			created_at
		) VALUES (?, ?, ?, ?, ?)
	`, s.messageDeletesTable)

	if err := s.session.Query(
		statement,
		projection.AccountID,
		projection.RoomID,
		projection.MessageSentAt.UTC(),
		projection.MessageID,
		projection.CreatedAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra message deletion projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) deleteRoomMemberRow(ctx context.Context, roomID, accountID string) error {
	if err := s.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE room_id = ? AND account_id = ?`, s.roomMembersTable),
		strings.TrimSpace(roomID),
		strings.TrimSpace(accountID),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("delete cassandra room member projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) deleteRoomRow(ctx context.Context, roomID string) error {
	if err := s.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE room_id = ?`, s.roomTable),
		strings.TrimSpace(roomID),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("delete cassandra room projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) deleteRoomTimelinePartition(ctx context.Context, roomID string) error {
	if err := s.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE room_id = ?`, s.roomTimelineTable),
		strings.TrimSpace(roomID),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("delete cassandra room timeline projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) deleteMessageDeletePartition(ctx context.Context, accountID, roomID string) error {
	if err := s.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE account_id = ? AND room_id = ?`, s.messageDeletesTable),
		strings.TrimSpace(accountID),
		strings.TrimSpace(roomID),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("delete cassandra message deletion projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) getRoomRow(ctx context.Context, roomID string) (*roomProjectionRow, error) {
	statement := fmt.Sprintf(`
		SELECT
			room_id,
			name,
			description,
			room_type,
			owner_id,
			pinned_message_id,
			member_count,
			last_message_id,
			last_message_at,
			last_message_content,
			last_message_sender_id,
			created_at,
			updated_at
		FROM %s
		WHERE room_id = ?
	`, s.roomTable)

	row := &roomProjectionRow{}
	if err := s.session.Query(statement, strings.TrimSpace(roomID)).WithContext(ctx).Scan(
		&row.RoomID,
		&row.Name,
		&row.Description,
		&row.RoomType,
		&row.OwnerID,
		&row.PinnedMessageID,
		&row.MemberCount,
		&row.LastMessageID,
		&row.LastMessageAt,
		&row.LastMessageContent,
		&row.LastMessageSenderID,
		&row.CreatedAt,
		&row.UpdatedAt,
	); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(fmt.Errorf("get cassandra room projection failed: %v", err))
	}

	row.CreatedAt = row.CreatedAt.UTC()
	row.UpdatedAt = row.UpdatedAt.UTC()
	row.LastMessageAt = cloneTime(row.LastMessageAt)
	return row, nil
}

func (s *cassandraProjectionStore) upsertRoomRow(ctx context.Context, row *roomProjectionRow) error {
	statement := fmt.Sprintf(`
		INSERT INTO %s (
			room_id,
			name,
			description,
			room_type,
			owner_id,
			pinned_message_id,
			member_count,
			last_message_id,
			last_message_at,
			last_message_content,
			last_message_sender_id,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.roomTable)

	if err := s.session.Query(
		statement,
		row.RoomID,
		row.Name,
		row.Description,
		row.RoomType,
		row.OwnerID,
		nullableProjectionString(row.PinnedMessageID),
		row.MemberCount,
		nullableProjectionString(row.LastMessageID),
		row.LastMessageAt,
		nullableProjectionString(row.LastMessageContent),
		nullableProjectionString(row.LastMessageSenderID),
		row.CreatedAt.UTC(),
		row.UpdatedAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra room projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) syncRoomIndexes(ctx context.Context, previous, current *roomProjectionRow) error {
	if current == nil {
		return nil
	}

	members, err := s.listRoomMemberRows(ctx, current.RoomID)
	if err != nil {
		return stackErr.Error(err)
	}

	if previous != nil && !previous.UpdatedAt.Equal(current.UpdatedAt) {
		for _, member := range members {
			if err := s.deleteAccountRoomIndex(ctx, strings.TrimSpace(member.AccountID), previous); err != nil {
				return stackErr.Error(err)
			}
		}
	}

	for _, member := range members {
		if err := s.upsertAccountRoomIndex(ctx, strings.TrimSpace(member.AccountID), current); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func (s *cassandraProjectionStore) upsertAccountRoomIndex(ctx context.Context, accountID string, room *roomProjectionRow) error {
	statement := fmt.Sprintf(`
		INSERT INTO %s (
			account_id,
			room_updated_at,
			room_id,
			name,
			description,
			room_type,
			owner_id,
			pinned_message_id,
			member_count,
			last_message_id,
			last_message_at,
			last_message_content,
			last_message_sender_id,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.roomsByAccountTable)

	if err := s.session.Query(
		statement,
		accountID,
		room.UpdatedAt.UTC(),
		room.RoomID,
		room.Name,
		room.Description,
		room.RoomType,
		room.OwnerID,
		nullableProjectionString(room.PinnedMessageID),
		room.MemberCount,
		nullableProjectionString(room.LastMessageID),
		room.LastMessageAt,
		nullableProjectionString(room.LastMessageContent),
		nullableProjectionString(room.LastMessageSenderID),
		room.CreatedAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra room-by-account projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) deleteAccountRoomIndex(ctx context.Context, accountID string, room *roomProjectionRow) error {
	if room == nil {
		return nil
	}
	if err := s.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE account_id = ? AND room_updated_at = ? AND room_id = ?`, s.roomsByAccountTable),
		accountID,
		room.UpdatedAt.UTC(),
		room.RoomID,
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("delete cassandra room-by-account projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) upsertTimelineMessageRow(ctx context.Context, projection *roomprojection.MessageProjection) error {
	mentionsJSON, err := json.Marshal(projection.Mentions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal cassandra timeline mentions failed: %v", err))
	}

	statement := fmt.Sprintf(`
		INSERT INTO %s (
			room_id,
			message_sent_at,
			message_id,
			room_name,
			room_type,
			message_content,
			message_type,
			reply_to_message_id,
			forwarded_from_message_id,
			file_name,
			file_size,
			mime_type,
			object_key,
			message_sender_id,
			message_sender_name,
			message_sender_email,
			mentions_json,
			mention_all,
			mentioned_account_ids,
			edited_at,
			deleted_for_everyone_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.roomTimelineTable)

	if err := s.session.Query(
		statement,
		projection.RoomID,
		projection.MessageSentAt.UTC(),
		projection.MessageID,
		projection.RoomName,
		projection.RoomType,
		projection.MessageContent,
		projection.MessageType,
		nullableProjectionString(projection.ReplyToMessageID),
		nullableProjectionString(projection.ForwardedFromMessageID),
		nullableProjectionString(projection.FileName),
		projection.FileSize,
		nullableProjectionString(projection.MimeType),
		nullableProjectionString(projection.ObjectKey),
		projection.MessageSenderID,
		nullableProjectionString(projection.MessageSenderName),
		nullableProjectionString(projection.MessageSenderEmail),
		string(mentionsJSON),
		projection.MentionAll,
		projection.MentionedAccountIDs,
		projection.EditedAt,
		projection.DeletedForEveryoneAt,
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra room timeline projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) upsertMessageByIDRow(ctx context.Context, projection *roomprojection.MessageProjection) error {
	mentionsJSON, err := json.Marshal(projection.Mentions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal cassandra message-by-id mentions failed: %v", err))
	}

	statement := fmt.Sprintf(`
		INSERT INTO %s (
			message_id,
			room_id,
			room_name,
			room_type,
			message_content,
			message_type,
			reply_to_message_id,
			forwarded_from_message_id,
			file_name,
			file_size,
			mime_type,
			object_key,
			message_sender_id,
			message_sender_name,
			message_sender_email,
			message_sent_at,
			mentions_json,
			mention_all,
			mentioned_account_ids,
			edited_at,
			deleted_for_everyone_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.messageByIDTable)

	if err := s.session.Query(
		statement,
		projection.MessageID,
		projection.RoomID,
		projection.RoomName,
		projection.RoomType,
		projection.MessageContent,
		projection.MessageType,
		nullableProjectionString(projection.ReplyToMessageID),
		nullableProjectionString(projection.ForwardedFromMessageID),
		nullableProjectionString(projection.FileName),
		projection.FileSize,
		nullableProjectionString(projection.MimeType),
		nullableProjectionString(projection.ObjectKey),
		projection.MessageSenderID,
		nullableProjectionString(projection.MessageSenderName),
		nullableProjectionString(projection.MessageSenderEmail),
		projection.MessageSentAt.UTC(),
		string(mentionsJSON),
		projection.MentionAll,
		projection.MentionedAccountIDs,
		projection.EditedAt,
		projection.DeletedForEveryoneAt,
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra room message-by-id projection failed: %v", err))
	}
	return nil
}

func (s *cassandraProjectionStore) listTimelineBatch(ctx context.Context, roomID string, beforeAt *time.Time, limit int, ascending bool) ([]*messageProjectionRow, error) {
	operator := "<"
	order := ""
	args := []interface{}{roomID}

	if ascending {
		order = " ORDER BY message_sent_at ASC, message_id ASC"
	}

	statement := fmt.Sprintf(`
		SELECT
			room_id,
			room_name,
			room_type,
			message_id,
			message_content,
			message_type,
			reply_to_message_id,
			forwarded_from_message_id,
			file_name,
			file_size,
			mime_type,
			object_key,
			message_sender_id,
			message_sender_name,
			message_sender_email,
			message_sent_at,
			mentions_json,
			mention_all,
			mentioned_account_ids,
			edited_at,
			deleted_for_everyone_at
		FROM %s
		WHERE room_id = ?
	`, s.roomTimelineTable)

	if beforeAt != nil {
		statement += fmt.Sprintf(" AND message_sent_at %s ?", operator)
		args = append(args, beforeAt.UTC())
	}
	statement += order + " LIMIT ?"
	args = append(args, limit)

	return s.scanMessageRows(ctx, statement, args...)
}

func (s *cassandraProjectionStore) listUnreadTimelineBatch(ctx context.Context, roomID string, afterAt *time.Time, limit int) ([]*messageProjectionRow, error) {
	statement := fmt.Sprintf(`
		SELECT
			room_id,
			room_name,
			room_type,
			message_id,
			message_content,
			message_type,
			reply_to_message_id,
			forwarded_from_message_id,
			file_name,
			file_size,
			mime_type,
			object_key,
			message_sender_id,
			message_sender_name,
			message_sender_email,
			message_sent_at,
			mentions_json,
			mention_all,
			mentioned_account_ids,
			edited_at,
			deleted_for_everyone_at
		FROM %s
		WHERE room_id = ?
	`, s.roomTimelineTable)

	args := []interface{}{roomID}
	if afterAt != nil {
		statement += " AND message_sent_at > ?"
		args = append(args, afterAt.UTC())
	}
	statement += " LIMIT ?"
	args = append(args, limit)

	return s.scanMessageRows(ctx, statement, args...)
}

func (s *cassandraProjectionStore) scanMessageRows(ctx context.Context, statement string, args ...interface{}) ([]*messageProjectionRow, error) {
	rows := make([]*messageProjectionRow, 0)
	iter := s.session.Query(statement, args...).WithContext(ctx).Iter()
	defer iter.Close()

	scanner := iter.Scanner()
	for scanner.Next() {
		row := &messageProjectionRow{}
		if err := scanner.Scan(
			&row.RoomID,
			&row.RoomName,
			&row.RoomType,
			&row.MessageID,
			&row.MessageContent,
			&row.MessageType,
			&row.ReplyToMessageID,
			&row.ForwardedFromMessageID,
			&row.FileName,
			&row.FileSize,
			&row.MimeType,
			&row.ObjectKey,
			&row.MessageSenderID,
			&row.MessageSenderName,
			&row.MessageSenderEmail,
			&row.MessageSentAt,
			&row.MentionsJSON,
			&row.MentionAll,
			&row.MentionedAccountIDs,
			&row.EditedAt,
			&row.DeletedForEveryoneAt,
		); err != nil {
			return nil, stackErr.Error(fmt.Errorf("scan cassandra timeline projection failed: %v", err))
		}
		row.MessageSentAt = row.MessageSentAt.UTC()
		row.EditedAt = cloneTime(row.EditedAt)
		row.DeletedForEveryoneAt = cloneTime(row.DeletedForEveryoneAt)
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate cassandra timeline projections failed: %v", err))
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("close cassandra timeline iterator failed: %v", err))
	}
	return rows, nil
}

func (s *cassandraProjectionStore) listDeletedMessageIDs(ctx context.Context, accountID, roomID string, from, to *time.Time) (map[string]struct{}, error) {
	if accountID == "" || roomID == "" || from == nil || to == nil {
		return map[string]struct{}{}, nil
	}

	statement := fmt.Sprintf(`
		SELECT message_id
		FROM %s
		WHERE account_id = ? AND room_id = ? AND message_sent_at >= ? AND message_sent_at <= ?
	`, s.messageDeletesTable)

	iter := s.session.Query(statement, accountID, roomID, from.UTC(), to.UTC()).WithContext(ctx).Iter()
	defer iter.Close()

	results := make(map[string]struct{})
	scanner := iter.Scanner()
	var messageID string
	for scanner.Next() {
		if err := scanner.Scan(&messageID); err != nil {
			return nil, stackErr.Error(fmt.Errorf("scan cassandra message deletion projection failed: %v", err))
		}
		results[strings.TrimSpace(messageID)] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate cassandra message deletions failed: %v", err))
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("close cassandra message deletion iterator failed: %v", err))
	}
	return results, nil
}

func roomRowToEntity(row *roomProjectionRow) *views.RoomView {
	if row == nil {
		return nil
	}

	pinnedMessageID := stringPtr(row.PinnedMessageID)
	lastMessageID := stringPtr(row.LastMessageID)
	lastMessageContent := stringPtr(row.LastMessageContent)
	lastMessageSenderID := stringPtr(row.LastMessageSenderID)

	return &views.RoomView{
		ID:                  row.RoomID,
		Name:                row.Name,
		Description:         row.Description,
		RoomType:            row.RoomType,
		OwnerID:             row.OwnerID,
		PinnedMessageID:     pinnedMessageID,
		MemberCount:         row.MemberCount,
		LastMessageID:       lastMessageID,
		LastMessageAt:       cloneTime(row.LastMessageAt),
		LastMessageContent:  lastMessageContent,
		LastMessageSenderID: lastMessageSenderID,
		CreatedAt:           row.CreatedAt.UTC(),
		UpdatedAt:           row.UpdatedAt.UTC(),
	}
}

func roomMemberRowToEntity(row *roomMemberProjectionRow) *views.RoomMemberView {
	if row == nil {
		return nil
	}
	return &views.RoomMemberView{
		ID:              row.MemberID,
		RoomID:          row.RoomID,
		AccountID:       row.AccountID,
		DisplayName:     strings.TrimSpace(row.DisplayName),
		Username:        strings.TrimSpace(row.Username),
		AvatarObjectKey: strings.TrimSpace(row.AvatarObjectKey),
		Role:            row.Role,
		LastDeliveredAt: cloneTime(row.LastDeliveredAt),
		LastReadAt:      cloneTime(row.LastReadAt),
		CreatedAt:       row.CreatedAt.UTC(),
		UpdatedAt:       row.UpdatedAt.UTC(),
	}
}

func messageRowToEntity(row *messageProjectionRow) (*views.MessageView, error) {
	if row == nil {
		return nil, nil
	}

	mentions, err := unmarshalProjectionMentions(row.MentionsJSON)
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
		MentionAll:             row.MentionAll,
		ReplyToMessageID:       strings.TrimSpace(row.ReplyToMessageID),
		ForwardedFromMessageID: strings.TrimSpace(row.ForwardedFromMessageID),
		FileName:               strings.TrimSpace(row.FileName),
		FileSize:               row.FileSize,
		MimeType:               strings.TrimSpace(row.MimeType),
		ObjectKey:              strings.TrimSpace(row.ObjectKey),
		EditedAt:               cloneTime(row.EditedAt),
		DeletedForEveryoneAt:   cloneTime(row.DeletedForEveryoneAt),
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
		row.LastMessageAt = cloneTime(projection.LastMessage.MessageSentAt)
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
		LastDeliveredAt: cloneTime(projection.LastDeliveredAt),
		LastReadAt:      cloneTime(projection.LastReadAt),
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
		MentionAll:             message.MentionAll,
		MentionedAccountIDs:    mentionedAccountIDs,
		EditedAt:               cloneTime(message.EditedAt),
		DeletedForEveryoneAt:   cloneTime(message.DeletedForEveryoneAt),
	}
}

func roomLastMessageFromView(room *views.RoomView) *roomprojection.RoomLastMessageProjection {
	if room == nil || room.LastMessageID == nil {
		return nil
	}
	return &roomprojection.RoomLastMessageProjection{
		MessageID:       strings.TrimSpace(*room.LastMessageID),
		MessageSentAt:   cloneTime(room.LastMessageAt),
		MessageContent:  derefProjectionString(room.LastMessageContent),
		MessageSenderID: derefProjectionString(room.LastMessageSenderID),
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

func roomIDFromMessageView(message *views.MessageView) string {
	if message == nil {
		return ""
	}
	return strings.TrimSpace(message.RoomID)
}

func derefProjectionString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func cloneRoomRow(row *roomProjectionRow) *roomProjectionRow {
	if row == nil {
		return nil
	}
	copy := *row
	copy.LastMessageAt = cloneTime(row.LastMessageAt)
	return &copy
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func nullableProjectionString(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
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

func sliceRoomEntities(rows []*roomProjectionRow, offset, limit int) []*views.RoomView {
	if offset >= len(rows) {
		return []*views.RoomView{}
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}

	results := make([]*views.RoomView, 0, end-offset)
	for _, row := range rows[offset:end] {
		results = append(results, roomRowToEntity(row))
	}
	return results
}

func (s *cassandraProjectionStore) listRoomsFromBaseProjection(ctx context.Context, options utils.QueryOptions) ([]*views.RoomView, error) {
	limit, offset := normalizeOffsetLimit(options.Limit, options.Offset, 20, 100)

	statement := fmt.Sprintf(`
		SELECT
			room_id,
			name,
			description,
			room_type,
			owner_id,
			pinned_message_id,
			member_count,
			last_message_id,
			last_message_at,
			last_message_content,
			last_message_sender_id,
			created_at,
			updated_at
		FROM %s
	`, s.roomTable)

	rows := make([]*roomProjectionRow, 0)
	iter := s.session.Query(statement).WithContext(ctx).Iter()
	defer iter.Close()

	var (
		roomID              string
		name                string
		description         string
		roomType            string
		ownerID             string
		pinnedMessageID     string
		memberCount         int
		lastMessageID       string
		lastMessageAt       *time.Time
		lastMessageContent  string
		lastMessageSenderID string
		createdAt           time.Time
		updatedAt           time.Time
	)
	scanner := iter.Scanner()
	for scanner.Next() {
		if err := scanner.Scan(
			&roomID,
			&name,
			&description,
			&roomType,
			&ownerID,
			&pinnedMessageID,
			&memberCount,
			&lastMessageID,
			&lastMessageAt,
			&lastMessageContent,
			&lastMessageSenderID,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, stackErr.Error(fmt.Errorf("scan cassandra room projection failed: %v", err))
		}
		rows = append(rows, &roomProjectionRow{
			RoomID:              roomID,
			Name:                name,
			Description:         description,
			RoomType:            roomType,
			OwnerID:             ownerID,
			PinnedMessageID:     pinnedMessageID,
			MemberCount:         memberCount,
			LastMessageID:       lastMessageID,
			LastMessageAt:       cloneTime(lastMessageAt),
			LastMessageContent:  lastMessageContent,
			LastMessageSenderID: lastMessageSenderID,
			CreatedAt:           createdAt.UTC(),
			UpdatedAt:           updatedAt.UTC(),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate cassandra room projections failed: %v", err))
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("close cassandra room projection iterator failed: %v", err))
	}

	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].UpdatedAt.Equal(rows[j].UpdatedAt) {
			return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
		}
		return rows[i].RoomID > rows[j].RoomID
	})

	return sliceRoomEntities(rows, offset, limit), nil
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

func limitWithOffset(limit, offset int) int {
	value := limit + offset
	if value <= 0 {
		return limit
	}
	return value
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

func normalizeProjectionTable(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "room_projection_default"
	}
	return value
}

func marshalProjectionMentions(mentions []roomprojection.ProjectionMention) (string, error) {
	if len(mentions) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(mentions)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return string(data), nil
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
