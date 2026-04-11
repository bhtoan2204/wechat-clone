package entity

import (
	"errors"
	"strings"
	"time"

	roomtypes "go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/stackErr"
)

var (
	ErrRoomMemberIDRequired      = errors.New("member id is required")
	ErrRoomMemberAccountRequired = errors.New("account_id is required")
	ErrRoomMemberRoomRequired    = errors.New("room_id is required")
	ErrRoomMemberRoleInvalid     = errors.New("role is invalid")
	ErrRoomReceiptStatusInvalid  = errors.New("status must be delivered or seen")
)

func NewRoomMember(id, roomID, accountID string, role roomtypes.RoomRole, now time.Time) (*RoomMemberEntity, error) {
	id = strings.TrimSpace(id)
	roomID = strings.TrimSpace(roomID)
	accountID = strings.TrimSpace(accountID)
	role = normalizeRoomRole(role)

	switch {
	case id == "":
		return nil, ErrRoomMemberIDRequired
	case roomID == "":
		return nil, ErrRoomMemberRoomRequired
	case accountID == "":
		return nil, ErrRoomMemberAccountRequired
	case !role.IsValid():
		return nil, ErrRoomMemberRoleInvalid
	}

	now = normalizeRoomTime(now)
	return &RoomMemberEntity{
		ID:        id,
		RoomID:    roomID,
		AccountID: accountID,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func BuildGroupMemberRoles(ownerID string, memberIDs []string) (map[string]roomtypes.RoomRole, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, ErrRoomOwnerRequired
	}

	memberSet := map[string]roomtypes.RoomRole{ownerID: roomtypes.RoomRoleOwner}
	for _, memberID := range memberIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			continue
		}
		if _, exists := memberSet[memberID]; !exists {
			memberSet[memberID] = roomtypes.RoomRoleMember
		}
	}
	return memberSet, nil
}

func (m *RoomMemberEntity) IsManager() bool {
	if m == nil {
		return false
	}
	role := normalizeRoomRole(m.Role)
	return role == roomtypes.RoomRoleOwner || role == roomtypes.RoomRoleAdmin
}

func (m *RoomMemberEntity) CanManageGroup(room *Room) error {
	if m == nil {
		return ErrRoomMemberRequired
	}
	if err := room.RequireGroup(); err != nil {
		return stackErr.Error(err)
	}
	if !m.IsManager() {
		return ErrRoomInsufficientPermission
	}
	return nil
}

func (m *RoomMemberEntity) CanRemoveFrom(room *Room, targetAccountID string) error {
	if m == nil {
		return ErrRoomMemberRequired
	}
	if err := room.RequireGroup(); err != nil {
		return stackErr.Error(err)
	}

	targetAccountID = strings.TrimSpace(targetAccountID)
	if targetAccountID == "" {
		return ErrRoomMemberAccountRequired
	}
	if m.AccountID != targetAccountID && !m.IsManager() {
		return ErrRoomInsufficientPermission
	}
	if m.AccountID == targetAccountID && normalizeRoomRole(m.Role) == roomtypes.RoomRoleOwner {
		return ErrRoomOwnerCannotLeave
	}
	return nil
}

func (m *RoomMemberEntity) ApplyReceiptStatus(status string, updatedAt time.Time) (string, *time.Time, *time.Time, error) {
	normalizedStatus, err := NormalizeReceiptStatus(status)
	if err != nil {
		return "", nil, nil, stackErr.Error(err)
	}

	now := normalizeRoomTime(updatedAt)
	deliveredAt := &now
	var seenAt *time.Time

	m.LastDeliveredAt = deliveredAt
	if normalizedStatus == "seen" {
		seenAt = &now
		m.LastReadAt = seenAt
	}
	m.UpdatedAt = now

	return normalizedStatus, deliveredAt, seenAt, nil
}

func normalizeRoomRole(role roomtypes.RoomRole) roomtypes.RoomRole {
	if normalized := role.Normalize(); normalized != "" {
		return normalized
	}
	return roomtypes.RoomRoleMember
}

func NormalizeReceiptStatus(status string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "delivered":
		return "delivered", nil
	case "seen":
		return "seen", nil
	default:
		return "", ErrRoomReceiptStatusInvalid
	}
}
