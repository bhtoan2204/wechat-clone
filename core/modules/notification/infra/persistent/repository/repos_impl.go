package repository

import (
	"context"
	"fmt"

	appCtx "wechat-clone/core/context"
	notificationquery "wechat-clone/core/modules/notification/application/query"
	"wechat-clone/core/modules/notification/domain/repos"
	"wechat-clone/core/modules/notification/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

type repoImpl struct {
	session *gocql.Session
	tables  models.TableNames

	notificationRepo     repos.NotificationRepository
	pushSubscriptionRepo repos.PushSubscriptionRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) (repos.Repos, error) {
	return newRepoImpl(appCtx.GetCassandraSession())
}

func NewNotificationReadRepository(session *gocql.Session) (notificationquery.NotificationReadRepository, error) {
	repo, err := newNotificationRepo(session)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return repo, nil
}

func newRepoImpl(session *gocql.Session) (repos.Repos, error) {
	if session == nil {
		return nil, stackErr.Error(fmt.Errorf("notification repository requires cassandra session"))
	}

	tables := models.DefaultTableNames()
	if err := runCassandraMigrations(context.Background(), session, tables); err != nil {
		return nil, stackErr.Error(err)
	}

	notificationRepo, err := newNotificationRepo(session)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &repoImpl{
		session:              session,
		tables:               tables,
		notificationRepo:     notificationRepo,
		pushSubscriptionRepo: NewPushSubscriptionRepoImpl(session, tables),
	}, nil
}

func (r *repoImpl) NotificationRepository() repos.NotificationRepository {
	return r.notificationRepo
}

func (r *repoImpl) PushSubscriptionRepository() repos.PushSubscriptionRepository {
	return r.pushSubscriptionRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(repos.Repos) error) error {
	if fn == nil {
		return nil
	}
	return stackErr.Error(fn(r))
}
