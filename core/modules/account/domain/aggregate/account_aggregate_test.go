package aggregate

import (
	"testing"
	"time"

	valueobject "go-socket/core/modules/account/domain/value_object"
	accounttypes "go-socket/core/modules/account/types"
)

func TestAccountAggregateRegister(t *testing.T) {
	agg, err := NewAccountAggregate("account-1")
	if err != nil {
		t.Fatalf("NewAccountAggregate() error = %v", err)
	}

	createdAt := time.Now().UTC()
	email, err := valueobject.NewEmail("user@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}
	passwordHash, err := valueobject.NewHashedPassword("hashed-password")
	if err != nil {
		t.Fatalf("NewHashedPassword() error = %v", err)
	}

	if err := agg.Register(email, passwordHash, "User", createdAt); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if agg.AccountID != "account-1" {
		t.Fatalf("AccountID = %q, want %q", agg.AccountID, "account-1")
	}
	if agg.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", agg.Email, "user@example.com")
	}
	if agg.PasswordHash != "hashed-password" {
		t.Fatalf("PasswordHash = %q, want %q", agg.PasswordHash, "hashed-password")
	}
	if agg.DisplayName != "User" {
		t.Fatalf("DisplayName = %q, want %q", agg.DisplayName, "User")
	}
	if agg.Status != accounttypes.AccountStatusActive {
		t.Fatalf("Status = %q, want %q", agg.Status, accounttypes.AccountStatusActive)
	}
	if !agg.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", agg.CreatedAt, createdAt)
	}
}

func TestAccountAggregateUpdateProfileKeepsNilFieldsUntouched(t *testing.T) {
	agg, err := NewAccountAggregate("account-1")
	if err != nil {
		t.Fatalf("NewAccountAggregate() error = %v", err)
	}

	email, err := valueobject.NewEmail("user@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}
	passwordHash, err := valueobject.NewHashedPassword("hashed-password")
	if err != nil {
		t.Fatalf("NewHashedPassword() error = %v", err)
	}
	if err := agg.Register(email, passwordHash, "User", time.Now().UTC()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	initialUsername := "first-user"
	initialAvatar := "avatar/object"
	if _, err := agg.UpdateProfile("User", &initialUsername, &initialAvatar, time.Now().UTC()); err != nil {
		t.Fatalf("UpdateProfile() setup error = %v", err)
	}

	updated, err := agg.UpdateProfile("Updated User", nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if !updated {
		t.Fatalf("UpdateProfile() updated = false, want true")
	}
	if agg.Username == nil || *agg.Username != initialUsername {
		t.Fatalf("Username = %v, want %q", agg.Username, initialUsername)
	}
	if agg.AvatarObjectKey == nil || *agg.AvatarObjectKey != initialAvatar {
		t.Fatalf("AvatarObjectKey = %v, want %q", agg.AvatarObjectKey, initialAvatar)
	}

	empty := ""
	updated, err = agg.UpdateProfile("Updated User", &empty, &empty, time.Now().UTC())
	if err != nil {
		t.Fatalf("UpdateProfile() clear error = %v", err)
	}
	if !updated {
		t.Fatalf("UpdateProfile() clear updated = false, want true")
	}
	if agg.Username != nil {
		t.Fatalf("Username = %v, want nil", agg.Username)
	}
	if agg.AvatarObjectKey != nil {
		t.Fatalf("AvatarObjectKey = %v, want nil", agg.AvatarObjectKey)
	}
}
