package read_repo

import (
	"context"
	"go-socket/core/modules/room/infra/projection/cassandra/views"

	"github.com/gocql/gocql"
)

type roomReadRepo struct {
	session *gocql.Session
}

func NewRoomReadRepo(session *gocql.Session) *roomReadRepo {
	return &roomReadRepo{session: session}
}

func (r *roomReadRepo) Upsert(ctx context.Context, room *views.RoomView) error {
	query := `
	INSERT INTO room_read_models (
		id,
		name,
		description,
		room_type,
		owner_id,
		direct_key,
		pinned_message_id,
		member_count,
		last_message_id,
		last_message_at,
		last_message_content,
		last_message_sender_id,
		created_at,
		updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return r.session.Query(query,
		room.ID,
		room.Name,
		room.Description,
		room.RoomType,
		room.OwnerID,
		room.DirectKey,
		room.PinnedMessageID,
		room.MemberCount,
		room.LastMessageID,
		room.LastMessageAt,
		room.LastMessageContent,
		room.LastMessageSenderID,
		room.CreatedAt,
		room.UpdatedAt,
	).WithContext(ctx).Exec()
}

func (r *roomReadRepo) GetByID(ctx context.Context, id string) (*views.RoomView, error) {
	query := `
	SELECT
		id,
		name,
		description,
		room_type,
		owner_id,
		direct_key,
		pinned_message_id,
		member_count,
		last_message_id,
		last_message_at,
		last_message_content,
		last_message_sender_id,
		created_at,
		updated_at
	FROM room_read_models
	WHERE id = ?
	LIMIT 1
	`

	var room views.RoomView

	if err := r.session.Query(query, id).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(
			&room.ID,
			&room.Name,
			&room.Description,
			&room.RoomType,
			&room.OwnerID,
			&room.DirectKey,
			&room.PinnedMessageID,
			&room.MemberCount,
			&room.LastMessageID,
			&room.LastMessageAt,
			&room.LastMessageContent,
			&room.LastMessageSenderID,
			&room.CreatedAt,
			&room.UpdatedAt,
		); err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &room, nil
}
