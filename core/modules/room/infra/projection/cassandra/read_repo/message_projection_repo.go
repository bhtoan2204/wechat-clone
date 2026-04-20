package read_repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	roomprojection "wechat-clone/core/modules/room/application/projection"
	"wechat-clone/core/modules/room/infra/projection/cassandra/views"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"github.com/gocql/gocql"
)

type MessageProjectionRow struct {
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
	ReactionsJSON          string
	MentionAll             bool
	MentionedAccountIDs    []string
	EditedAt               *time.Time
	DeletedForEveryoneAt   *time.Time
}

type MessageProjectionRepo struct {
	session           *gocql.Session
	roomTimelineTable string
	messageByIDTable  string
}

func NewMessageProjectionRepo(session *gocql.Session, tables views.ProjectionTableNames) *MessageProjectionRepo {
	return &MessageProjectionRepo{
		session:           session,
		roomTimelineTable: tables.MessageTimelines,
		messageByIDTable:  tables.MessageByID,
	}
}

func (r *MessageProjectionRepo) UpsertTimelineRow(ctx context.Context, projection *roomprojection.MessageProjection) error {
	mentionsJSON, err := json.Marshal(projection.Mentions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal cassandra timeline mentions failed: %w", err))
	}
	reactionsJSON, err := json.Marshal(projection.Reactions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal cassandra timeline reactions failed: %w", err))
	}
	statement := fmt.Sprintf(`INSERT INTO %s (room_id,message_sent_at,message_id,room_name,room_type,message_content,message_type,reply_to_message_id,forwarded_from_message_id,file_name,file_size,mime_type,object_key,message_sender_id,message_sender_name,message_sender_email,mentions_json,reactions_json,mention_all,mentioned_account_ids,edited_at,deleted_for_everyone_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, r.roomTimelineTable)
	return stackErr.Error(r.session.Query(statement, projection.RoomID, projection.MessageSentAt.UTC(), projection.MessageID, projection.RoomName, projection.RoomType, projection.MessageContent, projection.MessageType, nullableProjectionString(projection.ReplyToMessageID), nullableProjectionString(projection.ForwardedFromMessageID), nullableProjectionString(projection.FileName), projection.FileSize, nullableProjectionString(projection.MimeType), nullableProjectionString(projection.ObjectKey), projection.MessageSenderID, nullableProjectionString(projection.MessageSenderName), nullableProjectionString(projection.MessageSenderEmail), string(mentionsJSON), string(reactionsJSON), projection.MentionAll, projection.MentionedAccountIDs, projection.EditedAt, projection.DeletedForEveryoneAt).WithContext(ctx).Exec())
}

func (r *MessageProjectionRepo) UpsertByIDRow(ctx context.Context, projection *roomprojection.MessageProjection) error {
	mentionsJSON, err := json.Marshal(projection.Mentions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal cassandra message-by-id mentions failed: %w", err))
	}
	reactionsJSON, err := json.Marshal(projection.Reactions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal cassandra message-by-id reactions failed: %w", err))
	}
	statement := fmt.Sprintf(`INSERT INTO %s (message_id,room_id,room_name,room_type,message_content,message_type,reply_to_message_id,forwarded_from_message_id,file_name,file_size,mime_type,object_key,message_sender_id,message_sender_name,message_sender_email,message_sent_at,mentions_json,reactions_json,mention_all,mentioned_account_ids,edited_at,deleted_for_everyone_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, r.messageByIDTable)
	return stackErr.Error(r.session.Query(statement, projection.MessageID, projection.RoomID, projection.RoomName, projection.RoomType, projection.MessageContent, projection.MessageType, nullableProjectionString(projection.ReplyToMessageID), nullableProjectionString(projection.ForwardedFromMessageID), nullableProjectionString(projection.FileName), projection.FileSize, nullableProjectionString(projection.MimeType), nullableProjectionString(projection.ObjectKey), projection.MessageSenderID, nullableProjectionString(projection.MessageSenderName), nullableProjectionString(projection.MessageSenderEmail), projection.MessageSentAt.UTC(), string(mentionsJSON), string(reactionsJSON), projection.MentionAll, projection.MentionedAccountIDs, projection.EditedAt, projection.DeletedForEveryoneAt).WithContext(ctx).Exec())
}

func (r *MessageProjectionRepo) GetMessageByIDRow(ctx context.Context, id string) (*MessageProjectionRow, error) {
	statement := fmt.Sprintf(`SELECT room_id,room_name,room_type,message_id,message_content,message_type,reply_to_message_id,forwarded_from_message_id,file_name,file_size,mime_type,object_key,message_sender_id,message_sender_name,message_sender_email,message_sent_at,mentions_json,reactions_json,mention_all,mentioned_account_ids,edited_at,deleted_for_everyone_at FROM %s WHERE message_id = ?`, r.messageByIDTable)
	row := &MessageProjectionRow{}
	if err := r.session.Query(statement, strings.TrimSpace(id)).WithContext(ctx).Scan(&row.RoomID, &row.RoomName, &row.RoomType, &row.MessageID, &row.MessageContent, &row.MessageType, &row.ReplyToMessageID, &row.ForwardedFromMessageID, &row.FileName, &row.FileSize, &row.MimeType, &row.ObjectKey, &row.MessageSenderID, &row.MessageSenderName, &row.MessageSenderEmail, &row.MessageSentAt, &row.MentionsJSON, &row.ReactionsJSON, &row.MentionAll, &row.MentionedAccountIDs, &row.EditedAt, &row.DeletedForEveryoneAt); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}
	return row, nil
}

func (r *MessageProjectionRepo) GetLastMessageRow(ctx context.Context, roomID string) (*MessageProjectionRow, error) {
	statement := fmt.Sprintf(`SELECT room_id,room_name,room_type,message_id,message_content,message_type,reply_to_message_id,forwarded_from_message_id,file_name,file_size,mime_type,object_key,message_sender_id,message_sender_name,message_sender_email,message_sent_at,mentions_json,reactions_json,mention_all,mentioned_account_ids,edited_at,deleted_for_everyone_at FROM %s WHERE room_id = ? LIMIT 1`, r.roomTimelineTable)
	row := &MessageProjectionRow{}
	if err := r.session.Query(statement, strings.TrimSpace(roomID)).WithContext(ctx).Scan(&row.RoomID, &row.RoomName, &row.RoomType, &row.MessageID, &row.MessageContent, &row.MessageType, &row.ReplyToMessageID, &row.ForwardedFromMessageID, &row.FileName, &row.FileSize, &row.MimeType, &row.ObjectKey, &row.MessageSenderID, &row.MessageSenderName, &row.MessageSenderEmail, &row.MessageSentAt, &row.MentionsJSON, &row.ReactionsJSON, &row.MentionAll, &row.MentionedAccountIDs, &row.EditedAt, &row.DeletedForEveryoneAt); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}
	return row, nil
}

func (r *MessageProjectionRepo) ListTimelineBatch(ctx context.Context, roomID string, beforeAt *time.Time, limit int, ascending bool) ([]*MessageProjectionRow, error) {
	order := ""
	args := []interface{}{roomID}
	if ascending {
		order = " ORDER BY message_sent_at ASC, message_id ASC"
	}
	statement := fmt.Sprintf(`SELECT room_id,room_name,room_type,message_id,message_content,message_type,reply_to_message_id,forwarded_from_message_id,file_name,file_size,mime_type,object_key,message_sender_id,message_sender_name,message_sender_email,message_sent_at,mentions_json,reactions_json,mention_all,mentioned_account_ids,edited_at,deleted_for_everyone_at FROM %s WHERE room_id = ?`, r.roomTimelineTable)
	if beforeAt != nil {
		statement += " AND message_sent_at < ?"
		args = append(args, beforeAt.UTC())
	}
	statement += order + " LIMIT ?"
	args = append(args, limit)
	return r.scanMessageRows(ctx, statement, args...)
}

func (r *MessageProjectionRepo) ListUnreadTimelineBatch(ctx context.Context, roomID string, afterAt *time.Time, limit int) ([]*MessageProjectionRow, error) {
	statement := fmt.Sprintf(`SELECT room_id,room_name,room_type,message_id,message_content,message_type,reply_to_message_id,forwarded_from_message_id,file_name,file_size,mime_type,object_key,message_sender_id,message_sender_name,message_sender_email,message_sent_at,mentions_json,reactions_json,mention_all,mentioned_account_ids,edited_at,deleted_for_everyone_at FROM %s WHERE room_id = ?`, r.roomTimelineTable)
	args := []interface{}{roomID}
	if afterAt != nil {
		statement += " AND message_sent_at > ?"
		args = append(args, afterAt.UTC())
	}
	statement += " LIMIT ?"
	args = append(args, limit)
	return r.scanMessageRows(ctx, statement, args...)
}

func (r *MessageProjectionRepo) DeleteRoomTimelinePartition(ctx context.Context, roomID string) error {
	return stackErr.Error(r.session.Query(fmt.Sprintf(`DELETE FROM %s WHERE room_id = ?`, r.roomTimelineTable), strings.TrimSpace(roomID)).WithContext(ctx).Exec())
}

func (r *MessageProjectionRepo) scanMessageRows(ctx context.Context, statement string, args ...interface{}) ([]*MessageProjectionRow, error) {
	rows := make([]*MessageProjectionRow, 0)
	iter := r.session.Query(statement, args...).WithContext(ctx).Iter()
	defer iter.Close()
	scanner := iter.Scanner()
	for scanner.Next() {
		row := &MessageProjectionRow{}
		if err := scanner.Scan(&row.RoomID, &row.RoomName, &row.RoomType, &row.MessageID, &row.MessageContent, &row.MessageType, &row.ReplyToMessageID, &row.ForwardedFromMessageID, &row.FileName, &row.FileSize, &row.MimeType, &row.ObjectKey, &row.MessageSenderID, &row.MessageSenderName, &row.MessageSenderEmail, &row.MessageSentAt, &row.MentionsJSON, &row.ReactionsJSON, &row.MentionAll, &row.MentionedAccountIDs, &row.EditedAt, &row.DeletedForEveryoneAt); err != nil {
			return nil, stackErr.Error(fmt.Errorf("scan cassandra timeline projection failed: %w", err))
		}
		row.MessageSentAt = row.MessageSentAt.UTC()
		row.EditedAt = utils.ClonePtr(row.EditedAt)
		row.DeletedForEveryoneAt = utils.ClonePtr(row.DeletedForEveryoneAt)
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate cassandra timeline projections failed: %w", err))
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("close cassandra timeline iterator failed: %w", err))
	}
	return rows, nil
}
