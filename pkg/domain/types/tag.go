package types

import (
	"crypto/rand"
	"regexp"

	"github.com/m-mizutani/goerr/v2"
)

type TagID string

func (x TagID) String() string {
	return string(x)
}

func (x TagID) Validate() error {
	if x == EmptyTagID {
		return goerr.New("empty tag ID")
	}

	// Validate 8 characters of [a-zA-Z0-9]
	matched, err := regexp.MatchString(`^[a-zA-Z0-9]{8}$`, string(x))
	if err != nil {
		return goerr.Wrap(err, "failed to validate tag ID format")
	}
	if !matched {
		return goerr.New("invalid tag ID format", goerr.V("id", x))
	}

	return nil
}

func NewTagID() TagID {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const idLength = 8

	id := generateRandomString(charset, idLength)
	return TagID(id)
}

func generateRandomString(charset string, length int) string {
	b := make([]byte, length)
	charsetLen := len(charset)

	// Calculate the maximum index that can be uniformly distributed
	// to avoid modulo bias
	maxUniform := 256 - (256 % charsetLen)

	for i := 0; i < length; i++ {
		// Use rejection sampling to ensure uniform distribution
		for {
			var randomByte [1]byte
			if _, err := rand.Read(randomByte[:]); err != nil {
				panic("failed to generate random byte: " + err.Error())
			}

			// Reject values that would cause modulo bias
			if int(randomByte[0]) < maxUniform {
				b[i] = charset[int(randomByte[0])%charsetLen]
				break
			}
			// Loop again if the value would cause bias
		}
	}

	return string(b)
}

const EmptyTagID TagID = ""
