package converter

import (
	"crypto/sha256"
	"testing"

	assertions "github.com/stretchr/testify/assert"

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
	assert := assertions.New(t)
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

	assert.NotEqual(hash1.Sum(nil), hash2.Sum(nil), "different secret contents should produce different hashes")
}

func TestHashSecrets_UsingSameSecretTwiceChangesHash(t *testing.T) {
	assert := assertions.New(t)
	secretA := createSecret("A", map[string]string{
		"a": "1",
		"b": "2",
	})

	hash1 := sha256.New()
	hashSecrets([]*core.Secret{secretA}, hash1)

	hash2 := sha256.New()
	hashSecrets([]*core.Secret{secretA, secretA}, hash2)

	assert.NotEqual(hash1.Sum(nil), hash2.Sum(nil), "multiple uses of the same secret should change the hash")
}

func TestHashSecrets_UsingSameSecretTwiceIsNotEmptyString(t *testing.T) {
	assert := assertions.New(t)
	secretA := createSecret("A", map[string]string{
		"a": "1",
		"b": "2",
	})

	hash1 := sha256.New()
	hashSecrets([]*core.Secret{secretA, secretA}, hash1)
	hash2 := sha256.New()
	assert.NotEqual(hash1.Sum(nil), hash2.Sum(nil), "if the same secret is used twice, it should not result in a hash of an empty string")
}
