package hasher

import (
	"context"
	"encoding/base64"
	"fmt"
	"go-socket/core/shared/pkg/stackErr"
	"strings"

	"golang.org/x/crypto/argon2"
)

type hasherImpl struct {
	Salt    []byte
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

func NewHasher() (Hasher, error) {
	return newHasher()
}

func newHasher() (Hasher, error) {
	salt, err := genSalt(32)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &hasherImpl{
		Salt:    salt,
		Time:    1,
		Memory:  64 * 1024,
		Threads: 4,
		KeyLen:  32,
	}, nil
}

func (h *hasherImpl) Hash(ctx context.Context, value string) (string, error) {
	hashedBytes := argon2.IDKey([]byte(value), h.Salt, h.Time, h.Memory, h.Threads, h.KeyLen)

	encodedSalt := base64.RawStdEncoding.EncodeToString(h.Salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hashedBytes)

	return fmt.Sprintf("%s$%s", encodedSalt, encodedHash), nil
}

func (h *hasherImpl) Verify(ctx context.Context, val string, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 2 {
		return false, stackErr.Error(fmt.Errorf("invalid stored hash format"))
	}
	encodedSalt, encodedHash := parts[0], parts[1]

	salt, err := base64.RawStdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return false, stackErr.Error(fmt.Errorf("failed to decode salt: %v", err))
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false, stackErr.Error(fmt.Errorf("failed to decode hash: %v", err))
	}

	computedHash := argon2.IDKey([]byte(val), salt, h.Time, h.Memory, h.Threads, h.KeyLen)

	if len(computedHash) != len(expectedHash) {
		return false, nil
	}
	for i := 0; i < len(computedHash); i++ {
		if computedHash[i] != expectedHash[i] {
			return false, nil
		}
	}
	return true, nil
}
