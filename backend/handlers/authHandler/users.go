package auth

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type password struct {
	plainText *string
	Hash      []byte
}

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

// Should be called only after plaintext is validated
func (psswd *password) SetHash(plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}
	psswd.plainText = &plaintext
	psswd.Hash = hash
	return nil
}

func (psswd *password) Matches(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(psswd.Hash, []byte(plaintext))
	if err != nil {
		switch err {
		case bcrypt.ErrMismatchedHashAndPassword:
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}
