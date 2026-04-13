package projection

import (
	"strings"

	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

func NewCassandraTimelineProjector(cfg config.CassandraConfig, session *gocql.Session) (events.TimelineProjector, error) {
	store, err := NewCassandraProjectionStore(cfg, session)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return store, nil
}

func normalizeTimelineTable(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "room_message_timelines"
}
