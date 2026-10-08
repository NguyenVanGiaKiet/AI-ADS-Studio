package service

import (
	"errors"
	"testing"
)

func TestNewAuthServiceRequiresConfiguredCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "missing username", password: "correct-horse-battery"},
		{name: "missing password", username: "admin"},
		{name: "short password", username: "admin", password: "short"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewAuthService(test.username, test.password); err == nil {
				t.Fatal("NewAuthService() succeeded, want configuration error")
			}
		})
	}
}

func TestAuthServiceLoginAndLogout(t *testing.T) {
	auth, err := NewAuthService("admin", "correct-horse-battery")
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	if _, err := auth.LoginFrom("192.0.2.20:12345", "admin", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	token, err := auth.LoginFrom("192.0.2.20:12345", "admin", "correct-horse-battery")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !auth.IsAuthenticated(token) {
		t.Fatal("IsAuthenticated() = false after successful login")
	}

	auth.Logout(token)
	if auth.IsAuthenticated(token) {
		t.Fatal("IsAuthenticated() = true after logout")
	}
}

func TestAuthServiceLimitsFailedLoginAttemptsByIP(t *testing.T) {
	auth, err := NewAuthService("admin", "correct-horse-battery")
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	for attempt := 0; attempt < loginAttemptLimit; attempt++ {
		_, err := auth.LoginFrom("192.0.2.10:12345", "admin", "wrong-password")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("LoginFrom() attempt %d error = %v, want ErrInvalidCredentials", attempt+1, err)
		}
	}
	if _, err := auth.LoginFrom("192.0.2.10:54321", "admin", "correct-horse-battery"); !errors.Is(err, ErrLoginRateLimited) {
		t.Fatalf("LoginFrom() after failed attempts error = %v, want ErrLoginRateLimited", err)
	}
	if _, err := auth.LoginFrom("192.0.2.11:12345", "admin", "correct-horse-battery"); err != nil {
		t.Fatalf("LoginFrom() from another IP error = %v", err)
	}
}
