package repository

import (
	"context"

	"wechat-clone/core/modules/notification/infra/persistent/models"
	sharedcassandra "wechat-clone/core/shared/infra/cassandra"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

const cassandraMigrationSource = "file://migration/cassandra/notification"

func runCassandraMigrations(ctx context.Context, session *gocql.Session, tables models.TableNames) error {
	tool := sharedcassandra.NewMigrateTool()
	if err := tool.MigrateFromSource(ctx, session, tables.SchemaMigrations, cassandraMigrationSource); err != nil {
		return stackErr.Error(err)
	}
	return nil
}
