package projection

import (
	"context"
	"sort"
	"strings"

	"go-socket/core/modules/room/application/projection"
	"go-socket/core/modules/room/domain/entity"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

type queryRepoImpl struct {
	roomReadRepo       projection.RoomReadRepository
	messageReadRepo    projection.MessageReadRepository
	roomMemberReadRepo projection.RoomMemberReadRepository
}

func NewQueryRepoImpl(
	cfg config.CassandraConfig,
	session *gocql.Session,
) (projection.QueryRepos, error) {
	store, err := NewCassandraProjectionStore(cfg, session)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &queryRepoImpl{
		roomReadRepo:       store,
		messageReadRepo:    store,
		roomMemberReadRepo: &roomMemberQueryRepo{store: store},
	}, nil
}

func (r *queryRepoImpl) RoomReadRepository() projection.RoomReadRepository {
	return r.roomReadRepo
}

func (r *queryRepoImpl) MessageReadRepository() projection.MessageReadRepository {
	return r.messageReadRepo
}

func (r *queryRepoImpl) RoomMemberReadRepository() projection.RoomMemberReadRepository {
	return r.roomMemberReadRepo
}

type roomMemberQueryRepo struct {
	store       *cassandraProjectionStore
	accountRepo roomrepos.RoomAccountProjectionRepository
}

func (r *roomMemberQueryRepo) ListRoomMembers(ctx context.Context, roomID string) ([]*views.RoomMemberView, error) {
	members, err := r.store.ListRoomMembers(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return r.enrichMembers(ctx, members)
}

func (r *roomMemberQueryRepo) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error) {
	member, err := r.store.GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if member == nil {
		return nil, nil
	}

	members, err := r.enrichMembers(ctx, []*views.RoomMemberView{member})
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if len(members) == 0 {
		return nil, nil
	}
	return members[0], nil
}

func (r *roomMemberQueryRepo) SearchMentionCandidates(
	ctx context.Context,
	roomID,
	keyword,
	excludeAccountID string,
	limit int,
) ([]*views.MentionCandidateView, error) {
	members, err := r.ListRoomMembers(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
	excludeAccountID = strings.TrimSpace(excludeAccountID)

	results := make([]*views.MentionCandidateView, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}

		accountID := strings.TrimSpace(member.AccountID)
		if accountID == "" || accountID == excludeAccountID {
			continue
		}

		if normalizedKeyword != "" &&
			!strings.Contains(strings.ToLower(member.DisplayName), normalizedKeyword) &&
			!strings.Contains(strings.ToLower(member.Username), normalizedKeyword) &&
			!strings.Contains(strings.ToLower(accountID), normalizedKeyword) {
			continue
		}

		results = append(results, &views.MentionCandidateView{
			AccountID:       accountID,
			DisplayName:     strings.TrimSpace(member.DisplayName),
			Username:        strings.TrimSpace(member.Username),
			AvatarObjectKey: strings.TrimSpace(member.AvatarObjectKey),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		leftName := strings.ToLower(firstNonEmpty(results[i].DisplayName, results[i].AccountID))
		rightName := strings.ToLower(firstNonEmpty(results[j].DisplayName, results[j].AccountID))
		if leftName != rightName {
			return leftName < rightName
		}

		leftUsername := strings.ToLower(results[i].Username)
		rightUsername := strings.ToLower(results[j].Username)
		if leftUsername != rightUsername {
			return leftUsername < rightUsername
		}

		return results[i].AccountID < results[j].AccountID
	})

	limit = normalizeMentionLimit(limit)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (r *roomMemberQueryRepo) enrichMembers(ctx context.Context, members []*views.RoomMemberView) ([]*views.RoomMemberView, error) {
	if len(members) == 0 || r.accountRepo == nil {
		return members, nil
	}

	accountIDs := make([]string, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		accountIDs = append(accountIDs, strings.TrimSpace(member.AccountID))
	}

	accountProjections, err := r.accountRepo.ListByAccountIDs(ctx, accountIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := make(map[string]*entity.AccountEntity, len(accountProjections))
	for _, projectionItem := range accountProjections {
		if projectionItem == nil {
			continue
		}
		accountMap[strings.TrimSpace(projectionItem.AccountID)] = projectionItem
	}

	results := make([]*views.RoomMemberView, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}

		copyMember := *member
		if account := accountMap[strings.TrimSpace(member.AccountID)]; account != nil {
			copyMember.DisplayName = strings.TrimSpace(account.DisplayName)
			copyMember.Username = strings.TrimSpace(account.Username)
			copyMember.AvatarObjectKey = strings.TrimSpace(account.AvatarObjectKey)
		}
		results = append(results, &copyMember)
	}
	return results, nil
}

func normalizeMentionLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
