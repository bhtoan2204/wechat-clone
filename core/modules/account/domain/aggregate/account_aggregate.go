package aggregate

import (
	"errors"
	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/modules/account/domain/rules"
	valueobject "go-socket/core/modules/account/domain/value_object"
	accounttypes "go-socket/core/modules/account/types"
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
	"time"
)

type AccountAggregate struct {
	event.AggregateRoot

	AccountID                      string
	Email                          string
	DisplayName                    string
	Username                       *string
	AvatarObjectKey                *string
	Status                         accounttypes.AccountStatus
	PasswordHash                   string
	EmailVerifiedAt                *time.Time
	LastEmailVerificationRequested *time.Time
	LastLoginAt                    *time.Time
	PasswordChangedAt              *time.Time
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
	BannedReason                   string
	BannedUntil                    *time.Time
}

func (a *AccountAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventAccountCreated{},
		&EventAccountUpdated{},
		&EventAccountProfileUpdated{},
		&EventAccountEmailVerificationRequested{},
		&EventAccountEmailVerified{},
		&EventAccountPasswordChanged{},
		&EventAccountBanned{},
	)
}

func (a *AccountAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventAccountCreated:
		return a.onAccountCreated(e.AggregateID, data)
	case *EventAccountUpdated:
		return a.onAccountUpdated(data)
	case *EventAccountProfileUpdated:
		return a.onAccountProfileUpdated(data)
	case *EventAccountEmailVerificationRequested:
		return a.onAccountEmailVerificationRequested(data)
	case *EventAccountEmailVerified:
		return a.onAccountEmailVerified(data)
	case *EventAccountPasswordChanged:
		return a.onAccountPasswordChanged(data)
	case *EventAccountBanned:
		return a.onAccountBanned(data)
	default:
		return stackErr.Error(errors.New("unsupported event type"))
	}
}

func (a *AccountAggregate) onAccountCreated(aggregateID string, data *EventAccountCreated) error {
	a.AccountID = aggregateID
	a.Email = data.Email
	a.PasswordHash = data.PasswordHash
	a.DisplayName = data.DisplayName
	a.Status = data.Status
	a.CreatedAt = data.CreatedAt
	a.UpdatedAt = data.CreatedAt
	return nil
}

func (a *AccountAggregate) onAccountUpdated(data *EventAccountUpdated) error {
	a.Email = data.Email
	a.UpdatedAt = data.UpdatedAt
	return nil
}

func (a *AccountAggregate) onAccountProfileUpdated(data *EventAccountProfileUpdated) error {
	a.DisplayName = data.DisplayName
	a.Username = data.Username
	a.AvatarObjectKey = data.AvatarObjectKey
	a.UpdatedAt = data.UpdatedAt
	return nil
}

func (a *AccountAggregate) onAccountEmailVerificationRequested(data *EventAccountEmailVerificationRequested) error {
	requestedAt := data.RequestedAt
	a.LastEmailVerificationRequested = &requestedAt
	a.UpdatedAt = requestedAt
	return nil
}

func (a *AccountAggregate) onAccountEmailVerified(data *EventAccountEmailVerified) error {
	verifiedAt := data.EmailVerifiedAt
	a.EmailVerifiedAt = &verifiedAt
	a.UpdatedAt = verifiedAt
	return nil
}

func (a *AccountAggregate) onAccountPasswordChanged(data *EventAccountPasswordChanged) error {
	changedAt := data.PasswordChangedAt
	a.PasswordHash = data.PasswordHash
	a.PasswordChangedAt = &changedAt
	a.UpdatedAt = changedAt
	return nil
}

func (a *AccountAggregate) onAccountBanned(data *EventAccountBanned) error {
	a.BannedReason = data.BanReason
	a.BannedUntil = data.BanUntil
	return nil
}

func (a *AccountAggregate) Register(
	email valueobject.Email,
	passwordHash valueobject.HashedPassword,
	displayName string,
	now time.Time,
) error {
	if a.IsRegistered() {
		return stackErr.Error(rules.ErrAccountAlreadyRegistered)
	}

	normalizedDisplayName, err := rules.NormalizeDisplayName(displayName)
	if err != nil {
		return stackErr.Error(err)
	}
	createdAt := rules.NormalizeAccountTime(now)

	return a.ApplyChange(a, &EventAccountCreated{
		AccountID:    a.AggregateID(),
		Email:        email.Value(),
		PasswordHash: passwordHash.Value(),
		DisplayName:  normalizedDisplayName,
		Status:       accounttypes.AccountStatusActive,
		CreatedAt:    createdAt,
	})
}

func (a *AccountAggregate) UpdateProfile(displayName string, username, avatarObjectKey *string, now time.Time) (bool, error) {
	if !a.IsRegistered() {
		return false, stackErr.Error(rules.ErrAccountNotRegistered)
	}

	normalizedDisplayName, err := rules.NormalizeDisplayName(displayName)
	if err != nil {
		return false, stackErr.Error(err)
	}
	normalizedUsername := cloneOptionalString(a.Username)
	if username != nil {
		normalizedUsername = rules.NormalizeOptionalString(*username)
	}
	normalizedAvatarObjectKey := cloneOptionalString(a.AvatarObjectKey)
	if avatarObjectKey != nil {
		normalizedAvatarObjectKey = rules.NormalizeOptionalString(*avatarObjectKey)
	}

	if a.DisplayName == normalizedDisplayName &&
		rules.EqualOptionalString(a.Username, normalizedUsername) &&
		rules.EqualOptionalString(a.AvatarObjectKey, normalizedAvatarObjectKey) {
		return false, nil
	}

	updatedAt := rules.NormalizeAccountTime(now)
	if err := a.ApplyChange(a, &EventAccountProfileUpdated{
		AccountID:       a.AggregateID(),
		DisplayName:     normalizedDisplayName,
		Username:        normalizedUsername,
		AvatarObjectKey: normalizedAvatarObjectKey,
		UpdatedAt:       updatedAt,
	}); err != nil {
		return false, stackErr.Error(err)
	}

	return true, nil
}

func (a *AccountAggregate) RequestEmailVerification(token string, requestedAt time.Time) error {
	if err := a.EnsureEmailVerificationAllowed(); err != nil {
		return stackErr.Error(err)
	}

	return a.ApplyChange(a, &EventAccountEmailVerificationRequested{
		AccountID:         a.AggregateID(),
		Email:             a.Email,
		VerificationToken: token,
		RequestedAt:       rules.NormalizeAccountTime(requestedAt),
	})
}

func (a *AccountAggregate) EnsureEmailVerificationAllowed() error {
	if !a.IsRegistered() {
		return stackErr.Error(rules.ErrAccountNotRegistered)
	}
	if a.EmailVerifiedAt != nil {
		return stackErr.Error(rules.ErrAccountAlreadyVerified)
	}
	return nil
}

func (a *AccountAggregate) ConfirmEmailVerified(email valueobject.Email, verifiedAt time.Time) error {
	if !a.IsRegistered() {
		return stackErr.Error(rules.ErrAccountNotRegistered)
	}
	if a.EmailVerifiedAt != nil {
		return stackErr.Error(rules.ErrAccountAlreadyVerified)
	}
	if a.Email != email.Value() {
		return stackErr.Error(rules.ErrAccountEmailMismatch)
	}

	return a.ApplyChange(a, &EventAccountEmailVerified{
		AccountID:       a.AggregateID(),
		EmailVerifiedAt: rules.NormalizeAccountTime(verifiedAt),
	})
}

func (a *AccountAggregate) ChangePassword(passwordHash valueobject.HashedPassword, now time.Time) (bool, error) {
	if !a.IsRegistered() {
		return false, stackErr.Error(rules.ErrAccountNotRegistered)
	}
	if a.PasswordHash == passwordHash.Value() {
		return false, stackErr.Error(rules.ErrAccountPasswordSameAsOldOne)
	}

	changedAt := rules.NormalizeAccountTime(now)
	if err := a.ApplyChange(a, &EventAccountPasswordChanged{
		AccountID:         a.AggregateID(),
		PasswordHash:      passwordHash.Value(),
		PasswordChangedAt: changedAt,
	}); err != nil {
		return false, stackErr.Error(err)
	}

	return true, nil
}

func (a *AccountAggregate) Snapshot() (*entity.Account, error) {
	email, err := valueobject.NewEmail(a.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	passwordHash, err := valueobject.NewHashedPassword(a.PasswordHash)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	status, err := rules.NormalizeStatus(a.Status)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &entity.Account{
		ID:                a.AccountID,
		Email:             email,
		PasswordHash:      passwordHash,
		DisplayName:       a.DisplayName,
		Username:          cloneOptionalString(a.Username),
		AvatarObjectKey:   cloneOptionalString(a.AvatarObjectKey),
		Status:            status,
		EmailVerifiedAt:   cloneTime(a.EmailVerifiedAt),
		LastLoginAt:       cloneTime(a.LastLoginAt),
		PasswordChangedAt: cloneTime(a.PasswordChangedAt),
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
		BannedReason:      a.BannedReason,
		BannedUntil:       cloneTime(a.BannedUntil),
	}, nil
}

func (a *AccountAggregate) RestoreFromProjection(snapshot *entity.Account, version int) error {
	if snapshot == nil {
		return nil
	}

	a.AccountID = snapshot.ID
	a.Email = snapshot.Email.Value()
	a.PasswordHash = snapshot.PasswordHash.Value()
	a.DisplayName = snapshot.DisplayName
	a.Username = cloneOptionalString(snapshot.Username)
	a.AvatarObjectKey = cloneOptionalString(snapshot.AvatarObjectKey)
	a.Status = snapshot.Status
	a.EmailVerifiedAt = cloneTime(snapshot.EmailVerifiedAt)
	a.LastLoginAt = cloneTime(snapshot.LastLoginAt)
	a.PasswordChangedAt = cloneTime(snapshot.PasswordChangedAt)
	a.CreatedAt = snapshot.CreatedAt
	a.UpdatedAt = snapshot.UpdatedAt
	a.BannedReason = snapshot.BannedReason
	a.BannedUntil = cloneTime(snapshot.BannedUntil)
	a.SetInternal(snapshot.ID, version, version)
	return nil
}

func (a *AccountAggregate) MergeProjection(snapshot *entity.Account) {
	if snapshot == nil {
		return
	}

	if a.AccountID == "" {
		a.AccountID = snapshot.ID
	}
	if a.Email == "" {
		a.Email = snapshot.Email.Value()
	}
	if a.PasswordHash == "" {
		a.PasswordHash = snapshot.PasswordHash.Value()
	}
	if a.DisplayName == "" {
		a.DisplayName = snapshot.DisplayName
	}
	if a.Username == nil {
		a.Username = cloneOptionalString(snapshot.Username)
	}
	if a.AvatarObjectKey == nil {
		a.AvatarObjectKey = cloneOptionalString(snapshot.AvatarObjectKey)
	}
	if a.Status == "" {
		a.Status = snapshot.Status
	}
	if a.EmailVerifiedAt == nil {
		a.EmailVerifiedAt = cloneTime(snapshot.EmailVerifiedAt)
	}
	a.LastLoginAt = cloneTime(snapshot.LastLoginAt)
	if a.PasswordChangedAt == nil {
		a.PasswordChangedAt = cloneTime(snapshot.PasswordChangedAt)
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = snapshot.CreatedAt
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = snapshot.UpdatedAt
	}
	if a.BannedReason == "" {
		a.BannedReason = snapshot.BannedReason
	}
	if a.BannedUntil == nil {
		a.BannedUntil = cloneTime(snapshot.BannedUntil)
	}
}

func (a *AccountAggregate) CurrentPasswordHash() (valueobject.HashedPassword, error) {
	return valueobject.NewHashedPassword(a.PasswordHash)
}

func (a *AccountAggregate) IsRegistered() bool {
	return a.AccountID != "" && !a.CreatedAt.IsZero()
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
