package data

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Password struct {
	plainText *string
	Hash      []byte
}

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  Password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

var anonymousUser User

func AnonymousUser() *User {
	return &anonymousUser
}

func (user *User) IsAnonymous() bool {
	return user == &anonymousUser
}

// Should be called only after plaintext is validated
func (psswd *Password) SetHash(plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}
	psswd.plainText = &plaintext
	psswd.Hash = hash
	return nil
}

func (psswd *Password) Matches(plaintext string) (bool, error) {
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
