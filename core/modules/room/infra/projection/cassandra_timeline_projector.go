package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	roomprojection "go-socket/core/modules/room/application/projection"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

type cassandraTimelineProjector struct {
	session *gocql.Session
	table   string
}

func NewCassandraTimelineProjector(cfg config.CassandraConfig, session *gocql.Session) (roomprojection.TimelineProjector, error) {
	if !cfg.Enabled || session == nil {
		return nil, nil
	}

	projector := &cassandraTimelineProjector{
		session: session,
		table:   normalizeTimelineTable(cfg.RoomTimelineTable),
	}

	if err := projector.ensureSchema(context.Background()); err != nil {
		return nil, stackErr.Error(err)
	}

	return projector, nil
}

func (p *cassandraTimelineProjector) UpsertMessage(ctx context.Context, item *roomprojection.TimelineMessageProjection) error {
	if p == nil || p.session == nil || item == nil {
		return nil
	}

	mentionsJSON, err := json.Marshal(item.Mentions)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal timeline mentions failed: %v", err))
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
			mention_all,
			mentioned_account_ids,
			mentions_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.table)

	if err := p.session.Query(
		statement,
		item.RoomID,
		item.MessageSentAt,
		item.MessageID,
		item.RoomName,
		item.RoomType,
		item.MessageContent,
		item.MessageType,
		item.ReplyToMessageID,
		item.ForwardedFromMessageID,
		item.FileName,
		item.FileSize,
		item.MimeType,
		item.ObjectKey,
		item.MessageSenderID,
		item.MessageSenderName,
		item.MessageSenderEmail,
		item.MentionAll,
		item.MentionedAccountIDs,
		string(mentionsJSON),
	).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("upsert cassandra room timeline failed: %v", err))
	}

	return nil
}

func (p *cassandraTimelineProjector) ensureSchema(ctx context.Context) error {
	statement := fmt.Sprintf(`
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
			mention_all boolean,
			mentioned_account_ids list<text>,
			mentions_json text,
			PRIMARY KEY ((room_id), message_sent_at, message_id)
		) WITH CLUSTERING ORDER BY (message_sent_at DESC, message_id DESC)
	`, p.table)

	if err := p.session.Query(statement).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("ensure cassandra room timeline schema failed: %v", err))
	}
	return nil
}

func normalizeTimelineTable(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "room_message_timelines"
	}
	return value
}
