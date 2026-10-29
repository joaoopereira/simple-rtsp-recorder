package config

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	pbkdf2Iterations = 210_000
	minPasswordLen   = 8
)

// ErrWeakPassword is returned for admin passwords that are too short.
var ErrWeakPassword = fmt.Errorf("a palavra-passe tem de ter pelo menos %d caracteres", minPasswordLen)

// HashPassword returns "pbkdf2-sha256$<iterations>$<salt>$<hash>".
func HashPassword(password string) (string, error) {
	if utf8.RuneCountInString(password) < minPasswordLen {
		return "", ErrWeakPassword
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, 32)
	if err != nil {
		return "", err
	}
	enc := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", pbkdf2Iterations, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

// CheckPassword reports whether password matches a hash from HashPassword.
func CheckPassword(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	enc := base64.RawStdEncoding
	salt, err1 := enc.DecodeString(parts[2])
	want, err2 := enc.DecodeString(parts[3])
	if err != nil || err1 != nil || err2 != nil || iter < 1 || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// SetAdminPassword hashes and stores a new admin password.
func (s *Store) SetAdminPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	cfg := s.Get()
	cfg.AdminPasswordHash = hash
	return s.Update(cfg)
}

// ErrNoAdminPassword means no admin password was set yet (see set-admin-password).
var ErrNoAdminPassword = errors.New("palavra-passe de administrador não definida")

// CheckAdminPassword verifies the admin password.
func (s *Store) CheckAdminPassword(password string) error {
	hash := s.Get().AdminPasswordHash
	if hash == "" {
		return ErrNoAdminPassword
	}
	if !CheckPassword(hash, password) {
		return errors.New("palavra-passe errada")
	}
	return nil
}
