package hasher

import (
	"crypto/rand"
	"fmt"
	"go-socket/core/shared/pkg/stackErr"
)

func genSalt(keyLen uint32) ([]byte, error) {
	salt := make([]byte, keyLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, stackErr.Error(fmt.Errorf("failed to generate salt: %v", err))
	}
	return salt, nil
}
