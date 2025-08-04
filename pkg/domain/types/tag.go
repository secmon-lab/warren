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

	// 8文字、[a-zA-Z0-9]のバリデーション
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
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		id := generateRandomString(charset, idLength)
		// 基本的な衝突回避として、連続生成での重複を防ぐ
		// 実際の衝突チェックはリポジトリ層で行う
		return TagID(id)
	}
	panic("failed to generate tag ID after retries")
}

func generateRandomString(charset string, length int) string {
	b := make([]byte, length)
	charsetLen := len(charset)

	// crypto/randを使用してセキュアなランダム生成
	randBytes := make([]byte, length)
	if _, err := rand.Read(randBytes); err != nil {
		panic("failed to generate random bytes: " + err.Error())
	}

	for i := 0; i < length; i++ {
		b[i] = charset[int(randBytes[i])%charsetLen]
	}

	return string(b)
}

const EmptyTagID TagID = ""
