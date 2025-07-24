package data

import (
	"game-scouter-api/internal/validator"
	"regexp"
)

func ValidateName(v *validator.Validator, name string) {
	v.Assert(name == "", "name", "name should not be empty")
	v.Assert(len(name) > 500, "name", "name should never be more than 500 bytes")
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func ValidatePlaintext(v *validator.Validator, psswd string) {
	v.Assert(psswd != "", "password", "should not be empty")
	v.Assert(len([]byte(psswd)) < 72, "password", "length should be less than 72 bytes")
	v.Assert(len([]byte(psswd)) >= 8, "password", "length should be more than 7 bytes")
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Assert(email == "", "email", "should not be empty")
	v.Assert(emailRegex.MatchString(email), "email", "not in valid format")
}

func (user *User) Validate(v *validator.Validator) {
	ValidateName(v, user.Name)
	ValidateEmail(v, user.Email)
	ValidatePlaintext(v, *user.Password.plainText)
}
