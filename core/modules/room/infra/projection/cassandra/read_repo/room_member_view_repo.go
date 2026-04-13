package read_repo

import (
	"context"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"time"

	"github.com/gocql/gocql"
)

type roomMemberReadRepo struct {
	session *gocql.Session
}

func NewRoomMemberReadRepo(session *gocql.Session) *roomMemberReadRepo {
	return &roomMemberReadRepo{session: session}
}

func (r *roomMemberReadRepo) Upsert(ctx context.Context, rm *views.RoomMemberView) error {
	query := `
	INSERT INTO room_member_read_models (
		id,
		room_id,
		account_id,
		role,
		last_delivered_at,
		last_read_at,
		created_at,
		updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// đảm bảo updated_at luôn được set
	if rm.UpdatedAt.IsZero() {
		rm.UpdatedAt = time.Now()
	}

	return r.session.Query(query,
		rm.ID,
		rm.RoomID,
		rm.AccountID,
		rm.Role,
		rm.LastDeliveredAt,
		rm.LastReadAt,
		rm.CreatedAt,
		rm.UpdatedAt,
	).WithContext(ctx).Exec()
}

func (r *roomMemberReadRepo) GetByID(ctx context.Context, id string) (*views.RoomMemberView, error) {
	query := `
	SELECT
		id,
		room_id,
		account_id,
		role,
		last_delivered_at,
		last_read_at,
		created_at,
		updated_at
	FROM room_member_read_models
	WHERE id = ?
	LIMIT 1
	`

	var rm views.RoomMemberView

	err := r.session.Query(query, id).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(
			&rm.ID,
			&rm.RoomID,
			&rm.AccountID,
			&rm.Role,
			&rm.LastDeliveredAt,
			&rm.LastReadAt,
			&rm.CreatedAt,
			&rm.UpdatedAt,
		)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &rm, nil
}

func (r *roomMemberReadRepo) GetByRoomAndAccount(
	ctx context.Context,
	roomID string,
	accountID string,
) (*views.RoomMemberView, error) {

	query := `
	SELECT
		id,
		room_id,
		account_id,
		role,
		last_delivered_at,
		last_read_at,
		created_at,
		updated_at
	FROM room_member_read_models
	WHERE room_id = ? AND account_id = ?
	LIMIT 1
	`

	var rm views.RoomMemberView

	err := r.session.Query(query, roomID, accountID).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(
			&rm.ID,
			&rm.RoomID,
			&rm.AccountID,
			&rm.Role,
			&rm.LastDeliveredAt,
			&rm.LastReadAt,
			&rm.CreatedAt,
			&rm.UpdatedAt,
		)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &rm, nil
}
