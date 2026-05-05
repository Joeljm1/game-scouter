package data

import (
	"game-scouter-api/internal/validator"
	"regexp"
	"strings"
)

func ValidateName(v *validator.Validator, name string) {
	v.Assert(strings.TrimSpace(name) != "", "nameEmpty", "name should not be empty")
	v.Assert(len(name) < 500, "nameLong", "name should never be more than 500 bytes")
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func ValidatePlaintext(v *validator.Validator, psswd string) {
	v.Assert(strings.TrimSpace(psswd) != "", "passwordEmpty", "should not be empty")
	v.Assert(len([]byte(psswd)) < 72, "passwordLong", "length should be less than 72 bytes")
	v.Assert(len([]byte(psswd)) >= 8, "passwordShort", "length should be more than 7 bytes")
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Assert(email != "", "emailEmpty", "should not be empty")
	v.Assert(emailRegex.MatchString(email), "emailInvalid", "not in valid format")
	v.Assert(strings.Contains(email, "."), "EmailInvalid", "not in valid format")
}

// NOTE: Did password as sep arg to avoid hashing and then checking
// ie if err in email or name then avoid wasting cpu on hashing
func ValidateUser(v *validator.Validator, name string, email string, password string) *User {
	ValidateName(v, name)
	ValidateEmail(v, email)
	ValidatePlaintext(v, password)
	if v.Valid() {
		return &User{
			Name:  name,
			Email: email,
		}
	}
	return nil
}
