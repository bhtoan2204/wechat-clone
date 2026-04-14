package entity

import (
	"errors"
	"strings"
	"time"

	"go-socket/core/shared/pkg/stackErr"
)

type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusRevoked SessionStatus = "revoked"
	SessionStatusExpired SessionStatus = "expired"
)

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrSessionRevoked = errors.New("session revoked")
	ErrSessionExpired = errors.New("session expired")
)

type Session struct {
	ID               string
	AccountID        string
	DeviceID         string
	RefreshTokenHash string
	Status           SessionStatus
	IPAddress        *string
	UserAgent        *string
	LastActivityAt   *time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	RevokedReason    *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewSession(
	id string,
	accountID string,
	deviceID string,
	refreshTokenHash string,
	expiresAt time.Time,
	now time.Time,
	ipAddress string,
	userAgent string,
) (*Session, error) {
	id = strings.TrimSpace(id)
	accountID = strings.TrimSpace(accountID)
	deviceID = strings.TrimSpace(deviceID)
	refreshTokenHash = strings.TrimSpace(refreshTokenHash)

	if id == "" || accountID == "" || deviceID == "" || refreshTokenHash == "" || expiresAt.IsZero() {
		return nil, stackErr.Error(ErrInvalidSession)
	}

	normalizedNow := now.UTC()
	session := &Session{
		ID:               id,
		AccountID:        accountID,
		DeviceID:         deviceID,
		RefreshTokenHash: refreshTokenHash,
		Status:           SessionStatusActive,
		ExpiresAt:        expiresAt.UTC(),
		CreatedAt:        normalizedNow,
		UpdatedAt:        normalizedNow,
	}
	session.Touch(ipAddress, userAgent, normalizedNow)
	return session, nil
}

func (s *Session) EnsureRefreshAllowed(now time.Time) error {
	if s == nil {
		return stackErr.Error(ErrInvalidSession)
	}

	switch s.Status {
	case SessionStatusRevoked:
		return stackErr.Error(ErrSessionRevoked)
	case SessionStatusExpired:
		return stackErr.Error(ErrSessionExpired)
	}

	if !s.ExpiresAt.After(now.UTC()) {
		return stackErr.Error(ErrSessionExpired)
	}

	return nil
}

func (s *Session) Rotate(refreshTokenHash string, expiresAt time.Time, now time.Time, ipAddress string, userAgent string) error {
	if s == nil {
		return stackErr.Error(ErrInvalidSession)
	}

	refreshTokenHash = strings.TrimSpace(refreshTokenHash)
	if refreshTokenHash == "" || expiresAt.IsZero() {
		return stackErr.Error(ErrInvalidSession)
	}

	s.RefreshTokenHash = refreshTokenHash
	s.ExpiresAt = expiresAt.UTC()
	s.Status = SessionStatusActive
	s.RevokedAt = nil
	s.RevokedReason = nil
	s.Touch(ipAddress, userAgent, now)
	return nil
}

func (s *Session) Touch(ipAddress string, userAgent string, now time.Time) {
	if s == nil {
		return
	}

	if next := normalizeOptionalString(ipAddress); next != nil {
		s.IPAddress = next
	}
	if next := normalizeOptionalString(userAgent); next != nil {
		s.UserAgent = next
	}

	normalizedNow := now.UTC()
	s.LastActivityAt = &normalizedNow
	s.UpdatedAt = normalizedNow
	if s.CreatedAt.IsZero() {
		s.CreatedAt = normalizedNow
	}
}

func (s *Session) MarkExpired(now time.Time) bool {
	if s == nil || s.Status == SessionStatusRevoked || s.Status == SessionStatusExpired {
		return false
	}
	if s.ExpiresAt.After(now.UTC()) {
		return false
	}

	normalizedNow := now.UTC()
	s.Status = SessionStatusExpired
	s.UpdatedAt = normalizedNow
	return true
}

func (s *Session) Revoke(reason string, now time.Time) bool {
	if s == nil || s.Status == SessionStatusRevoked {
		return false
	}

	normalizedNow := now.UTC()
	s.Status = SessionStatusRevoked
	s.RevokedAt = &normalizedNow
	s.RevokedReason = normalizeOptionalString(reason)
	s.UpdatedAt = normalizedNow
	return true
}
