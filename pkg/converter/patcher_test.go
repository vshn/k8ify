package converter

import (
	"bytes"
	"crypto/sha256"
	"testing"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createSecret(name string, data map[string]string) *core.Secret {
	return &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		StringData: data,
	}
}

func TestHashSecrets_DifferentContentProducesDifferentHash(t *testing.T) {
	secretA := createSecret("A", map[string]string{
		"a": "1",
		"b": "2",
	})

	secretB := createSecret("B", map[string]string{
		"c": "3",
		"d": "4",
	})

	hash1 := sha256.New()
	hashSecrets([]*core.Secret{secretA}, hash1)

	hash2 := sha256.New()
	hashSecrets([]*core.Secret{secretB}, hash2)

	if bytes.Equal(hash1.Sum(nil), hash2.Sum(nil)) {
		t.Fatal("different secret contents should produce different hashes")
	}
}

func TestHashSecrets_OrderDoesNotMatter(t *testing.T) {
	secretA := createSecret("A", map[string]string{
		"a": "1",
		"b": "2",
	})

	secretB := createSecret("B", map[string]string{
		"c": "3",
		"d": "4",
	})

	hash1 := sha256.New()
	hashSecrets([]*core.Secret{secretA, secretB}, hash1)

	hash2 := sha256.New()
	hashSecrets([]*core.Secret{secretB, secretA}, hash2)

	if !bytes.Equal(hash1.Sum(nil), hash2.Sum(nil)) {
		t.Fatal("hashSecrets should produce identical hash regardless of slice order")
	}
}

func TestHashSecrets_UsingSameSecretTwiceChangesHash(t *testing.T) {
	secretA := createSecret("A", map[string]string{
		"a": "1",
		"b": "2",
	})

	hash1 := sha256.New()
	hashSecrets([]*core.Secret{secretA}, hash1)

	hash2 := sha256.New()
	hashSecrets([]*core.Secret{secretA, secretA}, hash2)

	if bytes.Equal(hash1.Sum(nil), hash2.Sum(nil)) {
		t.Fatal("multiple uses of the same secret should change the hash")
	}
}
