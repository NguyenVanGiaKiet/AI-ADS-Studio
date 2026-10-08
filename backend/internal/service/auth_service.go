package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

const AuthSessionTTL = 12 * time.Hour
const loginAttemptLimit = 5
const loginAttemptWindow = 15 * time.Minute

var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrLoginRateLimited = errors.New("too many login attempts")

type AuthService struct {
	usernameHash [sha256.Size]byte
	passwordHash [sha256.Size]byte
	mu           sync.Mutex
	sessions     map[string]time.Time
	attemptMu    sync.Mutex
	attempts     map[string]loginAttempt
}

type loginAttempt struct {
	failures int
	expires  time.Time
}

func NewAuthService(username, password string) (*AuthService, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("ADMIN_USERNAME must be set")
	}
	if len(password) < 12 {
		return nil, errors.New("ADMIN_PASSWORD must be at least 12 characters")
	}

	return &AuthService{
		usernameHash: sha256.Sum256([]byte(username)),
		passwordHash: sha256.Sum256([]byte(password)),
		sessions:     make(map[string]time.Time),
		attempts:     make(map[string]loginAttempt),
	}, nil
}

func (s *AuthService) LoginFrom(remoteAddress, username, password string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	}
	now := time.Now()

	s.attemptMu.Lock()
	for address, attempt := range s.attempts {
		if now.After(attempt.expires) {
			delete(s.attempts, address)
		}
	}
	attempt := s.attempts[remoteAddress]
	if attempt.failures >= loginAttemptLimit && now.Before(attempt.expires) {
		s.attemptMu.Unlock()
		return "", ErrLoginRateLimited
	}
	if !s.credentialsMatch(username, password) {
		attempt.failures++
		attempt.expires = now.Add(loginAttemptWindow)
		s.attempts[remoteAddress] = attempt
		s.attemptMu.Unlock()
		return "", ErrInvalidCredentials
	}
	delete(s.attempts, remoteAddress)
	s.attemptMu.Unlock()

	return s.createSession()
}

func (s *AuthService) credentialsMatch(username, password string) bool {
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	usernameMatches := subtle.ConstantTimeCompare(usernameHash[:], s.usernameHash[:])
	passwordMatches := subtle.ConstantTimeCompare(passwordHash[:], s.passwordHash[:])
	return usernameMatches&passwordMatches == 1
}

func (s *AuthService) createSession() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpiredSessions(time.Now())
	s.sessions[token] = time.Now().Add(AuthSessionTTL)
	return token, nil
}

func (s *AuthService) IsAuthenticated(token string) bool {
	if token == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expiresAt) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *AuthService) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *AuthService) removeExpiredSessions(now time.Time) {
	for token, expiresAt := range s.sessions {
		if now.After(expiresAt) {
			delete(s.sessions, token)
		}
	}
}
