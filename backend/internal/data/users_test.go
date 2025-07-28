package data

import (
	"crypto/rand"
	"game-scouter-api/internal/validator"
	"testing"
)

func TestIsAnonymous(t *testing.T) {
	tables := []struct {
		user     *User
		expected bool
	}{
		{ //1
			user:     &User{},
			expected: false,
		},
		{ //2
			user:     AnonymousUser(),
			expected: true,
		},
	}
	for i, table := range tables {
		result := table.user.IsAnonymous()
		if result != table.expected {
			t.Errorf("%v) IsAnonymous fn err . Expected %v, got %v", i+1, table.expected, result)
		}
	}
}

func TestValidateName(t *testing.T) {
	longName := make([]byte, 500)
	_, _ = rand.Read(longName)
	tables := []struct {
		name     string
		expected bool
	}{
		{ //1
			name:     "Joel",
			expected: true,
		},
		{ //2
			name:     "Jazael",
			expected: true,
		},
		{ //3
			name:     "",
			expected: false,
		},
		{ //4
			name:     string(longName),
			expected: false,
		},
		{ //5
			name:     "     ",
			expected: false,
		},
		{ //6
			name:     "\n\t\r",
			expected: false,
		},
		{ //7
			name:     "\t\n\v\f\r ",
			expected: false,
		},
	}
	for i, val := range tables {
		v := validator.NewValidator()
		ValidateName(v, val.name)
		if v.Valid() != val.expected {
			t.Errorf("%v) ValidateName err. Expected %v, got %v. Err: %v", i+1, val.expected, v.Valid(), v.Errors)
		}
	}
}

func TestValidPassword(t *testing.T) {
	longPass := make([]byte, 72)
	_, _ = rand.Read(longPass)
	tables := []struct {
		password string
		expected bool
	}{
		{ //1
			password: "123456789",
			expected: true,
		},
		{ //2
			password: "Hello123",
			expected: true,
		},
		{ //3
			password: "",
			expected: false,
		},
		{ //4
			password: string(longPass),
			expected: false,
		},
		{ //5
			password: "23e",
			expected: false,
		},
		{ //6
			password: "abc",
			expected: false,
		},
		{ //7
			password: "     ",
			expected: false,
		},
		{ //8
			password: "\n\t\r",
			expected: false,
		},
		{ //9
			password: "\t\n\v\f\r ",
			expected: false,
		},
	}
	for i, val := range tables {
		v := validator.NewValidator()
		ValidatePlaintext(v, val.password)
		if v.Valid() != val.expected {
			t.Errorf("%v) ValidatePassword err. Expected %v, got %v. Err: %v", i+1, val.expected, v.Valid(), v.Errors)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	table := []struct {
		email    string
		expected bool
	}{
		{
			email:    "132@yahoo.com",
			expected: true,
		},
		{
			email:    "joeljosep@gmail.com",
			expected: true,
		},
		{
			email:    "123@iiitk.ac.in",
			expected: true,
		},
		{
			email:    "",
			expected: false,
		},
		{
			email:    "2132gmail.com",
			expected: false,
		},
		{
			email:    "abc@yahoo",
			expected: false,
		},
		{
			email:    "joeljosep",
			expected: false,
		},
	}
	for i, val := range table {
		v := validator.NewValidator()
		ValidateEmail(v, val.email)
		if v.Valid() != val.expected {
			t.Errorf("%v) ValidateEmail err. Expected %v, got %v. Err: %v", i+1, val.expected, v.Valid(), v.Errors)
		}
	}
}
