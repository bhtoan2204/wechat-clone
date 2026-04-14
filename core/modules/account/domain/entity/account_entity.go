package entity

import (
	"go-socket/core/modules/account/domain/rules"
	valueobject "go-socket/core/modules/account/domain/value_object"
	accounttypes "go-socket/core/modules/account/types"
	"go-socket/core/shared/pkg/stackErr"
	"time"
)

type Account struct {
	ID                string                     `json:"id"`
	Email             valueobject.Email          `json:"email"`
	PasswordHash      valueobject.HashedPassword `json:"password_hash"`
	DisplayName       string                     `json:"display_name"`
	Username          *string                    `json:"username,omitempty"`
	AvatarObjectKey   *string                    `json:"avatar_object_key,omitempty"`
	Status            accounttypes.AccountStatus `json:"status"`
	EmailVerifiedAt   *time.Time                 `json:"email_verified_at,omitempty"`
	LastLoginAt       *time.Time                 `json:"last_login_at,omitempty"`
	PasswordChangedAt *time.Time                 `json:"password_changed_at,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
	BannedReason      string                     `json:"banned_reason"`
	BannedUntil       *time.Time                 `json:"banned_until"`
}

func NewAccount(
	id string,
	email valueobject.Email,
	passwordHash valueobject.HashedPassword,
	displayName string,
	status accounttypes.AccountStatus,
	now time.Time,
) (*Account, error) {
	normalizedID, err := rules.NormalizeAccountID(id)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	normalizedDisplayName, err := rules.NormalizeDisplayName(displayName)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	normalizedStatus, err := rules.NormalizeStatus(status)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	normalizedNow := rules.NormalizeAccountTime(now)
	return &Account{
		ID:           normalizedID,
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  normalizedDisplayName,
		Status:       normalizedStatus,
		CreatedAt:    normalizedNow,
		UpdatedAt:    normalizedNow,
	}, nil
}
