package policy

import (
	"fmt"

	"wechat-clone/core/modules/relationship/domain"
	"wechat-clone/core/shared/pkg/stackErr"
)

type PairState struct {
	ActorID             string
	TargetID            string
	TargetExists        bool
	IsFriend            bool
	IsFollowing         bool
	HasOutgoingRequest  bool
	HasIncomingRequest  bool
	HasBlockingRelation bool
	HasBlockedTarget    bool
}

func (s PairState) EnsureCanSendFriendRequest() error {
	if err := s.ensureTargetExists(); err != nil {
		return stackErr.Error(err)
	}
	if err := s.ensureNotSelf("send friend request to"); err != nil {
		return stackErr.Error(err)
	}
	if err := s.ensureNotBlocked(); err != nil {
		return stackErr.Error(err)
	}
	if s.IsFriend {
		return stackErr.Error(domain.ErrFriendshipAlreadyExists)
	}
	if s.HasOutgoingRequest || s.HasIncomingRequest {
		return stackErr.Error(domain.ErrFriendRequestAlreadyOpen)
	}
	return nil
}

func (s PairState) EnsureCanCancelFriendRequest() error {
	if err := s.ensureNotSelf("cancel friend request to"); err != nil {
		return stackErr.Error(err)
	}
	if !s.HasOutgoingRequest {
		return stackErr.Error(domain.ErrFriendRequestNotFound)
	}
	return nil
}

func (s PairState) EnsureCanAcceptFriendRequest() error {
	if err := s.ensureNotSelf("accept friend request from"); err != nil {
		return stackErr.Error(err)
	}
	if !s.HasIncomingRequest {
		return stackErr.Error(domain.ErrFriendRequestNotFound)
	}
	if err := s.ensureNotBlocked(); err != nil {
		return stackErr.Error(err)
	}
	if s.IsFriend {
		return stackErr.Error(domain.ErrFriendshipAlreadyExists)
	}
	return nil
}

func (s PairState) EnsureCanRejectFriendRequest() error {
	if err := s.ensureNotSelf("reject friend request from"); err != nil {
		return stackErr.Error(err)
	}
	if !s.HasIncomingRequest {
		return stackErr.Error(domain.ErrFriendRequestNotFound)
	}
	return nil
}

func (s PairState) EnsureCanFollow() error {
	if err := s.ensureTargetExists(); err != nil {
		return stackErr.Error(err)
	}
	if err := s.ensureNotSelf("follow"); err != nil {
		return stackErr.Error(err)
	}
	if err := s.ensureNotBlocked(); err != nil {
		return stackErr.Error(err)
	}
	if s.IsFollowing {
		return stackErr.Error(domain.ErrFollowAlreadyExists)
	}
	return nil
}

func (s PairState) EnsureCanUnfollow() error {
	if err := s.ensureNotSelf("unfollow"); err != nil {
		return stackErr.Error(err)
	}
	if !s.IsFollowing {
		return stackErr.Error(domain.ErrFollowNotFound)
	}
	return nil
}

func (s PairState) EnsureCanUnfriend() error {
	if err := s.ensureNotSelf("unfriend"); err != nil {
		return stackErr.Error(err)
	}
	if !s.IsFriend {
		return stackErr.Error(domain.ErrFriendshipNotFound)
	}
	return nil
}

func (s PairState) EnsureCanBlock() error {
	if err := s.ensureTargetExists(); err != nil {
		return stackErr.Error(err)
	}
	if err := s.ensureNotSelf("block"); err != nil {
		return stackErr.Error(err)
	}
	if s.HasBlockedTarget {
		return stackErr.Error(domain.ErrBlockAlreadyExists)
	}
	return nil
}

func (s PairState) EnsureCanUnblock() error {
	if err := s.ensureNotSelf("unblock"); err != nil {
		return stackErr.Error(err)
	}
	if !s.HasBlockedTarget {
		return stackErr.Error(domain.ErrBlockNotFound)
	}
	return nil
}

func (s PairState) ensureTargetExists() error {
	if !s.TargetExists {
		return stackErr.Error(domain.ErrTargetAccountNotFound)
	}
	return nil
}

func (s PairState) ensureNotSelf(action string) error {
	if s.ActorID == s.TargetID {
		return stackErr.Error(fmt.Errorf("cannot %s self", action))
	}
	return nil
}

func (s PairState) ensureNotBlocked() error {
	if s.HasBlockingRelation {
		return stackErr.Error(domain.ErrRelationshipBlocked)
	}
	return nil
}
