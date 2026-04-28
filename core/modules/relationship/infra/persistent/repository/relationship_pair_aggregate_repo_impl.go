package repository

import (
	"context"
	"errors"

	"wechat-clone/core/modules/relationship/domain"
	"wechat-clone/core/modules/relationship/domain/aggregate"
	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/modules/relationship/infra/persistent/models"
	dbinfra "wechat-clone/core/shared/infra/db"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type relationshipPairAggregateRepo struct {
	db                          *gorm.DB
	serializer                  eventpkg.Serializer
	outboxPublisher             eventpkg.Publisher
	friendRequestAggregateRepo  repos.FriendRequestAggregateRepository
	friendshipRepo              friendshipStore
	followRelationRepo          followRelationStore
	blockRelationRepo           blockRelationStore
	userRelationshipCounterRepo userRelationshipCounterStore
	accountProjectionRepo       relationshipAccountStore
	relationshipPairGuardRepo   relationshipPairGuardStore
}

func newRelationshipPairAggregateRepo(db *gorm.DB) repos.RelationshipPairAggregateRepository {
	serializer := eventpkg.NewSerializer()
	return &relationshipPairAggregateRepo{
		db:         db,
		serializer: serializer,
		outboxPublisher: eventpkg.NewPublisher(&relationOutboxEventStore{
			db:         db,
			serializer: serializer,
		}),
		friendRequestAggregateRepo:  newFriendRequestAggregateRepo(db),
		friendshipRepo:              newFriendshipRepo(db),
		followRelationRepo:          newFollowRelationRepo(db),
		blockRelationRepo:           newBlockRelationRepo(db),
		userRelationshipCounterRepo: newUserRelationshipCounterRepo(db),
		accountProjectionRepo:       newRelationshipAccountRepo(db),
		relationshipPairGuardRepo:   newRelationshipPairGuardRepo(db),
	}
}

func (r *relationshipPairAggregateRepo) LoadForUpdate(ctx context.Context, actorID, targetID string) (*aggregate.RelationshipPairAggregate, error) {
	if err := r.relationshipPairGuardRepo.LockPair(ctx, actorID, targetID); err != nil {
		return nil, stackErr.Error(err)
	}

	targetExists, err := r.accountProjectionRepo.Exists(ctx, targetID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	aggregateVersion, err := loadRelationOutboxAggregateVersion(
		r.db,
		aggregate.CanonicalRelationshipPairAggregateID(actorID, targetID),
		eventpkg.AggregateTypeName(&aggregate.RelationshipPairAggregate{}),
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	snapshot := aggregate.RelationshipPairSnapshot{
		ActorID:          actorID,
		TargetID:         targetID,
		TargetExists:     targetExists,
		FriendRequest:    nil,
		Friendship:       nil,
		Following:        nil,
		FollowedBy:       nil,
		Blocking:         nil,
		BlockedBy:        nil,
		AggregateVersion: aggregateVersion,
	}

	snapshot.FriendRequest, err = r.loadPendingFriendRequestBetween(ctx, actorID, targetID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	snapshot.Friendship, err = r.loadFriendshipBetween(ctx, actorID, targetID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	snapshot.Following, err = r.loadFollow(ctx, actorID, targetID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	snapshot.FollowedBy, err = r.loadFollow(ctx, targetID, actorID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	snapshot.Blocking, err = r.loadBlock(ctx, actorID, targetID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	snapshot.BlockedBy, err = r.loadBlock(ctx, targetID, actorID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := aggregate.NewRelationshipPair(snapshot)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *relationshipPairAggregateRepo) Save(ctx context.Context, agg *aggregate.RelationshipPairAggregate) error {
	if agg == nil {
		return stackErr.Error(errors.New("relationship pair aggregate is required"))
	}

	changes := agg.Changes()

	if err := r.persistFriendRequest(ctx, changes.FriendRequest); err != nil {
		return stackErr.Error(err)
	}
	if err := r.persistFriendship(ctx, agg, changes.Friendship); err != nil {
		return stackErr.Error(err)
	}
	if err := r.persistFollow(ctx, changes.Following); err != nil {
		return stackErr.Error(err)
	}
	if err := r.persistFollow(ctx, changes.FollowedBy); err != nil {
		return stackErr.Error(err)
	}
	if err := r.persistBlock(ctx, changes.Block); err != nil {
		return stackErr.Error(err)
	}
	if len(changes.CounterDeltas) > 0 {
		if err := r.userRelationshipCounterRepo.ApplyDeltas(ctx, changes.CounterDeltas); err != nil {
			return stackErr.Error(err)
		}
	}
	if err := r.publishOutboxEvents(ctx, agg); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *relationshipPairAggregateRepo) publishOutboxEvents(ctx context.Context, agg *aggregate.RelationshipPairAggregate) error {
	if agg == nil || len(agg.Events()) == 0 {
		return nil
	}
	if r == nil || r.outboxPublisher == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}
	if err := r.outboxPublisher.PublishAggregate(ctx, agg); err != nil {
		return stackErr.Error(err)
	}
	agg.MarkPersisted()
	return nil
}

func (r *relationshipPairAggregateRepo) persistFriendRequest(ctx context.Context, intent aggregate.FriendRequestPersistenceIntent) error {
	if intent.Kind != aggregate.RelationshipChangeUpsert || intent.Aggregate == nil {
		return nil
	}
	return stackErr.Error(r.friendRequestAggregateRepo.Save(ctx, intent.Aggregate))
}

func (r *relationshipPairAggregateRepo) persistFriendship(
	ctx context.Context,
	agg *aggregate.RelationshipPairAggregate,
	intent aggregate.FriendshipPersistenceIntent,
) error {
	switch intent.Kind {
	case aggregate.RelationshipChangeUpsert:
		if intent.Value == nil {
			return nil
		}
		if err := r.friendshipRepo.Create(ctx, intent.Value); err != nil {
			if dbinfra.IsUniqueConstraintError(err) {
				return stackErr.Error(domain.ErrFriendshipAlreadyExists)
			}
			return stackErr.Error(err)
		}
	case aggregate.RelationshipChangeDelete:
		deleted, err := r.friendshipRepo.DeleteBetween(ctx, agg.ActorID(), agg.TargetID())
		if err != nil {
			return stackErr.Error(err)
		}
		if !deleted {
			return stackErr.Error(domain.ErrFriendshipNotFound)
		}
	}
	return nil
}

func (r *relationshipPairAggregateRepo) persistFollow(ctx context.Context, intent aggregate.FollowPersistenceIntent) error {
	switch intent.Kind {
	case aggregate.RelationshipChangeUpsert:
		if intent.Value == nil {
			return nil
		}
		if err := r.followRelationRepo.Create(ctx, intent.Value); err != nil {
			if dbinfra.IsUniqueConstraintError(err) {
				return stackErr.Error(domain.ErrFollowAlreadyExists)
			}
			return stackErr.Error(err)
		}
	case aggregate.RelationshipChangeDelete:
		deleted, err := r.followRelationRepo.Delete(ctx, intent.FollowerID, intent.FolloweeID)
		if err != nil {
			return stackErr.Error(err)
		}
		if !deleted {
			return stackErr.Error(domain.ErrFollowNotFound)
		}
	}
	return nil
}

func (r *relationshipPairAggregateRepo) persistBlock(ctx context.Context, intent aggregate.BlockPersistenceIntent) error {
	switch intent.Kind {
	case aggregate.RelationshipChangeUpsert:
		if intent.Value == nil {
			return nil
		}
		if err := r.blockRelationRepo.Create(ctx, intent.Value); err != nil {
			if dbinfra.IsUniqueConstraintError(err) {
				return stackErr.Error(domain.ErrBlockAlreadyExists)
			}
			return stackErr.Error(err)
		}
	case aggregate.RelationshipChangeDelete:
		deleted, err := r.blockRelationRepo.Delete(ctx, intent.BlockerID, intent.BlockedID)
		if err != nil {
			return stackErr.Error(err)
		}
		if !deleted {
			return stackErr.Error(domain.ErrBlockNotFound)
		}
	}
	return nil
}

func (r *relationshipPairAggregateRepo) loadPendingFriendRequestBetween(ctx context.Context, actorID, targetID string) (*aggregate.FriendRequestAggregate, error) {
	friendRequest, err := r.friendRequestAggregateRepo.LoadPendingBetween(ctx, actorID, targetID)
	if err == nil {
		return friendRequest, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, stackErr.Error(err)
}

func (r *relationshipPairAggregateRepo) loadFriendshipBetween(ctx context.Context, actorID, targetID string) (*entity.Friendship, error) {
	userLowID, userHighID := normalizePair(actorID, targetID)

	var model models.Friendship
	err := r.db.WithContext(ctx).
		Where("user_low_id = ? AND user_high_id = ?", userLowID, userHighID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	return &entity.Friendship{
		ID:                   model.ID,
		UserLowID:            model.UserLowID,
		UserHighID:           model.UserHighID,
		CreatedAt:            model.CreatedAt,
		CreatedFromRequestID: model.CreatedFromRequestID,
	}, nil
}

func (r *relationshipPairAggregateRepo) loadFollow(ctx context.Context, followerID, followeeID string) (*entity.FollowRelation, error) {
	var model models.FollowRelation
	err := r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	return &entity.FollowRelation{
		ID:         model.ID,
		FollowerID: model.FollowerID,
		FolloweeID: model.FolloweeID,
		CreatedAt:  model.CreatedAt,
	}, nil
}

func (r *relationshipPairAggregateRepo) loadBlock(ctx context.Context, blockerID, blockedID string) (*entity.BlockRelation, error) {
	var model models.BlockRelation
	err := r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	return &entity.BlockRelation{
		ID:        model.ID,
		BlockerID: model.BlockerID,
		BlockedID: model.BlockedID,
		Reason:    model.Reason,
		CreatedAt: model.CreatedAt,
	}, nil
}
