package idgen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const (
	idLength        = 64
	idLengthInBytes = idLength / 2

	shortIDLength = 12
)

func ID() string {
	idBytes := make([]byte, idLengthInBytes)
	if _, err := rand.Read(idBytes); err != nil {
		panic(errors.New("failed to read random id bytes"))
	}

	hash := sha256.New()

	_, err := hash.Write(idBytes)
	if err != nil {
		panic(errors.New("failed to write sha256 hash"))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func ShortID(id string) string {
	if len(id) < shortIDLength {
		return id
	}

	return id[:shortIDLength]
}
