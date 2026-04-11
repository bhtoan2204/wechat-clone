package entity

import (
	"errors"
	"strings"
	"time"

	roomtypes "go-socket/core/modules/room/types"
)

var (
	ErrRoomIDRequired             = errors.New("id is required")
	ErrRoomNameRequired           = errors.New("name is required")
	ErrRoomOwnerRequired          = errors.New("account_id is required")
	ErrRoomTypeInvalid            = errors.New("room_type must be one of: public, private, direct, group")
	ErrRoomPeerAccountRequired    = errors.New("peer_account_id is required")
	ErrRoomDirectSelfNotAllowed   = errors.New("cannot create direct conversation with yourself")
	ErrRoomNotGroup               = errors.New("room is not a group")
	ErrRoomInsufficientPermission = errors.New("insufficient permissions")
	ErrRoomOwnerCannotLeave       = errors.New("owner cannot leave without transferring ownership")
	ErrRoomMessageIDRequired      = errors.New("message_id is required")
	ErrRoomMemberRequired         = errors.New("account is not a member of this room")
	ErrRoomMentionsRequireGroup   = errors.New("mentions are only supported in group rooms")
	ErrRoomMentionTargetNotMember = errors.New("mentioned account is not a member of this room")
)

func NewRoom(id, name, description, ownerID string, roomType roomtypes.RoomType, directKey string, now time.Time) (*Room, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	ownerID = strings.TrimSpace(ownerID)
	roomType = roomType.Normalize()
	directKey = strings.TrimSpace(directKey)

	switch {
	case id == "":
		return nil, ErrRoomIDRequired
	case name == "":
		return nil, ErrRoomNameRequired
	case ownerID == "":
		return nil, ErrRoomOwnerRequired
	case !roomType.IsValid():
		return nil, ErrRoomTypeInvalid
	}

	now = normalizeRoomTime(now)
	return &Room{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(description),
		OwnerID:     ownerID,
		RoomType:    roomType,
		DirectKey:   directKey,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func NewDirectConversationRoom(id, ownerID, peerID string, now time.Time) (*Room, error) {
	ownerID = strings.TrimSpace(ownerID)
	peerID = strings.TrimSpace(peerID)

	switch {
	case ownerID == "":
		return nil, ErrRoomOwnerRequired
	case peerID == "":
		return nil, ErrRoomPeerAccountRequired
	case ownerID == peerID:
		return nil, ErrRoomDirectSelfNotAllowed
	}

	return NewRoom(id, "Direct chat", "", ownerID, roomtypes.RoomTypeDirect, CanonicalDirectKey(ownerID, peerID), now)
}

func CanonicalDirectKey(a, b string) string {
	ids := []string{strings.TrimSpace(a), strings.TrimSpace(b)}
	if ids[0] > ids[1] {
		ids[0], ids[1] = ids[1], ids[0]
	}
	return strings.Join(ids, ":")
}

func (r *Room) IsGroup() bool {
	return r != nil && r.RoomType.Normalize() == roomtypes.RoomTypeGroup
}

func (r *Room) IsDirect() bool {
	return r != nil && r.RoomType.Normalize() == roomtypes.RoomTypeDirect
}

func (r *Room) UpdateDetails(name, description string, roomType roomtypes.RoomType, updatedAt time.Time) (bool, error) {
	if r == nil {
		return false, ErrRoomIDRequired
	}

	updated := false
	if name = strings.TrimSpace(name); name != "" && name != r.Name {
		r.Name = name
		updated = true
	}

	description = strings.TrimSpace(description)
	if description != r.Description {
		r.Description = description
		updated = true
	}

	if roomType = roomType.Normalize(); roomType != "" {
		if !roomType.IsValid() {
			return false, ErrRoomTypeInvalid
		}
		if roomType != r.RoomType {
			r.RoomType = roomType
			updated = true
		}
	}

	if updated {
		r.UpdatedAt = normalizeRoomTime(updatedAt)
	}
	return updated, nil
}

func (r *Room) RequireGroup() error {
	if !r.IsGroup() {
		return ErrRoomNotGroup
	}
	return nil
}

func (r *Room) PinMessage(messageID string, updatedAt time.Time) error {
	if strings.TrimSpace(messageID) == "" {
		return ErrRoomMessageIDRequired
	}
	r.PinnedMessageID = strings.TrimSpace(messageID)
	r.UpdatedAt = normalizeRoomTime(updatedAt)
	return nil
}

func (r *Room) Touch(updatedAt time.Time) {
	if r == nil {
		return
	}
	r.UpdatedAt = normalizeRoomTime(updatedAt)
}

func normalizeRoomTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
