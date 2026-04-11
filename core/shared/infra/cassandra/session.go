package cassandra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

func NewSession(ctx context.Context, cfg config.CassandraConfig) (*gocql.Session, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	hosts := splitAddresses(cfg.Hosts)
	if len(hosts) == 0 {
		return nil, stackErr.Error(fmt.Errorf("cassandra hosts are required when cassandra is enabled"))
	}

	baseCluster := newClusterConfig(cfg, hosts)
	baseSession, err := baseCluster.CreateSession()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("create cassandra bootstrap session failed: %v", err))
	}
	defer baseSession.Close()

	if err := ensureKeyspace(ctx, baseSession, cfg); err != nil {
		return nil, stackErr.Error(err)
	}

	cluster := newClusterConfig(cfg, hosts)
	cluster.Keyspace = strings.TrimSpace(cfg.Keyspace)

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("create cassandra session failed: %v", err))
	}
	return session, nil
}

func newClusterConfig(cfg config.CassandraConfig, hosts []string) *gocql.ClusterConfig {
	cluster := gocql.NewCluster(hosts...)
	if cfg.Port > 0 {
		cluster.Port = cfg.Port
	}
	cluster.Timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	cluster.ConnectTimeout = time.Duration(cfg.ConnectTimeoutSeconds) * time.Second
	cluster.Consistency = parseConsistency(cfg.Consistency)
	cluster.DisableInitialHostLookup = false

	if username := strings.TrimSpace(cfg.Username); username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: username,
			Password: cfg.Password,
		}
	}

	if localDC := strings.TrimSpace(cfg.LocalDC); localDC != "" {
		cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.DCAwareRoundRobinPolicy(localDC))
	}

	return cluster
}

func ensureKeyspace(ctx context.Context, session *gocql.Session, cfg config.CassandraConfig) error {
	if session == nil {
		return nil
	}

	keyspace := strings.TrimSpace(cfg.Keyspace)
	if keyspace == "" {
		return stackErr.Error(fmt.Errorf("cassandra keyspace is required"))
	}

	replicationClass := strings.TrimSpace(cfg.ReplicationClass)
	if replicationClass == "" {
		replicationClass = "SimpleStrategy"
	}
	replicationFactor := cfg.ReplicationFactor
	if replicationFactor <= 0 {
		replicationFactor = 1
	}

	statement := fmt.Sprintf(
		"CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': '%s', 'replication_factor': %d}",
		keyspace,
		replicationClass,
		replicationFactor,
	)
	if err := session.Query(statement).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("ensure cassandra keyspace failed: %v", err))
	}
	return nil
}

func parseConsistency(value string) gocql.Consistency {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all":
		return gocql.All
	case "one":
		return gocql.One
	case "two":
		return gocql.Two
	case "three":
		return gocql.Three
	case "localquorum":
		return gocql.LocalQuorum
	default:
		return gocql.Quorum
	}
}

func splitAddresses(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}
