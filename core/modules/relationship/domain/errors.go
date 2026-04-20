package domain

import "errors"

var (
	ErrNotFound                 = errors.New("not found")
	ErrDuplicate                = errors.New("duplicate value")
	ErrEmpty                    = errors.New("empty value")
	ErrInvalidData              = errors.New("invalid data")
	ErrTargetAccountNotFound    = errors.New("target account not found")
	ErrRelationshipBlocked      = errors.New("relationship blocked")
	ErrFriendRequestNotFound    = errors.New("friend request not found")
	ErrFriendRequestAlreadyOpen = errors.New("friend request already pending")
	ErrFriendshipAlreadyExists  = errors.New("friendship already exists")
	ErrFriendshipNotFound       = errors.New("friendship not found")
	ErrFollowAlreadyExists      = errors.New("follow relation already exists")
	ErrFollowNotFound           = errors.New("follow relation not found")
	ErrBlockAlreadyExists       = errors.New("block relation already exists")
	ErrBlockNotFound            = errors.New("block relation not found")
)
