package hash

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// BCrypt implements the Hasher interface by using BCrypt.
type BCrypt struct {
	WorkFactor int
}

func (b *BCrypt) Hash(data []byte) ([]byte, error) {
	s, err := bcrypt.GenerateFromPassword(data, b.WorkFactor)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return s, nil
}

func (b *BCrypt) Compare(hash, data []byte) error {
	if err := bcrypt.CompareHashAndPassword(hash, data); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}
