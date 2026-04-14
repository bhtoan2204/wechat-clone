package hasher

import "context"

//go:generate mockgen -package=hasher -destination=hasher_mock.go -source=hasher.go
type Hasher interface {
	Hash(ctx context.Context, value string) (string, error)
	Verify(ctx context.Context, val string, hash string) (bool, error)
}
