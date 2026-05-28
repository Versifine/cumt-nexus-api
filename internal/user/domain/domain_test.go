package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/google/uuid"
)

func TestNewUserID(t *testing.T) {
	raw := uuid.NewString()

	id, err := NewUserID(" " + raw + " ")
	if err != nil {
		t.Fatalf("NewUserID returned error: %v", err)
	}
	if id.String() != raw {
		t.Fatalf("expected normalized id %q, got %q", raw, id.String())
	}

	if _, err := NewUserID(""); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for empty user id, got %v", err)
	}
	if _, err := NewUserID("not-a-uuid"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid user id, got %v", err)
	}
}

func TestNewGeneratedUserID(t *testing.T) {
	id := NewGeneratedUserID()

	if _, err := NewUserID(id.String()); err != nil {
		t.Fatalf("generated user id should be valid: %v", err)
	}
}

func TestNewUsername(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "normalizes trim and case", raw: " Alice_123 ", want: "alice_123"},
		{name: "allows max length", raw: strings.Repeat("a", 20), want: strings.Repeat("a", 20)},
		{name: "rejects empty", raw: "   ", wantErr: true},
		{name: "rejects too short", raw: "ab", wantErr: true},
		{name: "rejects too long", raw: strings.Repeat("a", 21), wantErr: true},
		{name: "rejects illegal character", raw: "alice!", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := NewUsername(tt.raw)
			if tt.wantErr {
				if !hasAppCode(err, apperr.CodeInvalidArgument) {
					t.Fatalf("expected invalid_argument, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewUsername returned error: %v", err)
			}
			if username.String() != tt.want {
				t.Fatalf("expected username %q, got %q", tt.want, username.String())
			}
		})
	}
}

func TestPasswordValues(t *testing.T) {
	plainRaw := " password123 "
	plain, err := NewPlainPassword(plainRaw)
	if err != nil {
		t.Fatalf("NewPlainPassword returned error: %v", err)
	}
	if plain.String() != plainRaw {
		t.Fatalf("plain password should preserve user input")
	}

	if _, err := NewPlainPassword("        "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank password, got %v", err)
	}
	if _, err := NewPlainPassword("1234567"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for short password, got %v", err)
	}

	hash, err := NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	if hash.Raw() != "hashed-password" {
		t.Fatalf("expected password hash raw value, got %q", hash.Raw())
	}

	if _, err := NewPasswordHash("   "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank hash, got %v", err)
	}
}

func TestUserStatusAndCanLogin(t *testing.T) {
	activeStatus, err := NewUserStatus("active")
	if err != nil {
		t.Fatalf("NewUserStatus active returned error: %v", err)
	}
	if activeStatus.String() != "active" {
		t.Fatalf("expected active status string, got %q", activeStatus.String())
	}

	disabledStatus, err := NewUserStatus("disabled")
	if err != nil {
		t.Fatalf("NewUserStatus disabled returned error: %v", err)
	}

	now := time.Now().UTC()
	user := mustNewUser(t, now)
	if !user.CanLogin() {
		t.Fatal("new active user should be able to login")
	}

	disabledUser, err := RehydrateUser(user.ID(), user.Username(), user.PasswordHash(), disabledStatus, now, now)
	if err != nil {
		t.Fatalf("RehydrateUser disabled returned error: %v", err)
	}
	if disabledUser.CanLogin() {
		t.Fatal("disabled user should not be able to login")
	}

	if _, err := NewUserStatus("locked"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid status, got %v", err)
	}
}

func TestUserCreationAndRehydrateInvariants(t *testing.T) {
	if _, err := NewUser(mustUserID(t), mustUsername(t, "alice"), mustPasswordHash(t), time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero creation time, got %v", err)
	}

	now := time.Now().UTC()
	user := mustNewUser(t, now)
	if user.Status().String() != "active" {
		t.Fatalf("new user should default to active, got %q", user.Status().String())
	}
	if !user.CreatedAt().Equal(now) || !user.UpdatedAt().Equal(now) {
		t.Fatalf("new user timestamps should both equal creation time")
	}

	if _, err := RehydrateUser(user.ID(), user.Username(), user.PasswordHash(), user.Status(), time.Time{}, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero created_at, got %v", err)
	}
	if _, err := RehydrateUser(user.ID(), user.Username(), user.PasswordHash(), user.Status(), now, time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero updated_at, got %v", err)
	}
	if _, err := RehydrateUser(user.ID(), user.Username(), user.PasswordHash(), user.Status(), now, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for updated_at before created_at, got %v", err)
	}
}

func mustNewUser(t *testing.T, now time.Time) *User {
	t.Helper()

	user, err := NewUser(mustUserID(t), mustUsername(t, "alice"), mustPasswordHash(t), now)
	if err != nil {
		t.Fatalf("NewUser returned error: %v", err)
	}
	return user
}

func mustUserID(t *testing.T) UserID {
	t.Helper()

	id, err := NewUserID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewUserID returned error: %v", err)
	}
	return id
}

func mustUsername(t *testing.T, raw string) Username {
	t.Helper()

	username, err := NewUsername(raw)
	if err != nil {
		t.Fatalf("NewUsername returned error: %v", err)
	}
	return username
}

func mustPasswordHash(t *testing.T) PasswordHash {
	t.Helper()

	hash, err := NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	return hash
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}
