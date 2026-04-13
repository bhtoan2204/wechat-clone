package xpaseto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)

	fmt.Println("PUBLIC =", pubB64)
	fmt.Println("PRIVATE =", privB64)

	fmt.Println(len(pub))
	fmt.Println(len(priv))
}
