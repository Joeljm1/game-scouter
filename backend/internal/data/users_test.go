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
		{
			user:     &User{},
			expected: false,
		},
		{
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
	rand.Read(longName)
	tables := []struct {
		name     string
		expected bool
	}{
		{
			name:     "Joel",
			expected: true,
		},
		{
			name:     "Jazael",
			expected: true,
		},
		{
			name:     "",
			expected: false,
		},
		{
			name:     string(longName),
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
	rand.Read(longPass)
	tables := []struct {
		password string
		expected bool
	}{
		{
			password: "123456789",
			expected: true,
		},
		{
			password: "Hello123",
			expected: true,
		},
		{
			password: "",
			expected: false,
		},
		{
			password: string(longPass),
			expected: false,
		},
		{
			password: "23e",
			expected: false,
		},
		{
			password: "abc",
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
