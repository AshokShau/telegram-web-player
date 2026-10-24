package webapp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

type partnerInvite struct {
	roomID    int64
	expiresAt time.Time
	userLimit int
	users     map[int64]bool
	password  [32]byte
}

type partnerInviteStore struct {
	mu       sync.Mutex
	links    map[string]*partnerInvite
	attempts map[int64]struct {
		count int
		until time.Time
	}
}

var partnerInvites = &partnerInviteStore{links: make(map[string]*partnerInvite)}

func (s *partnerInviteStore) create(roomID int64, expiresAt time.Time, userLimit int) (string, error) {
	if expiresAt.Unix() == 0 {
		expiresAt = time.Time{}
	}
	if roomID <= 0 || (!expiresAt.IsZero() && !expiresAt.After(time.Now())) || userLimit < 0 {
		return "", errors.New("Choose a future expiry or no expiry, and a user limit of 0 or more.")
	}
	var random [18]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	token := "pm_" + strconv.FormatInt(roomID, 10) + "_" + base64.RawURLEncoding.EncodeToString(random[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, invite := range s.links {
		if !invite.expiresAt.IsZero() && !invite.expiresAt.After(time.Now()) {
			delete(s.links, key)
			continue
		}
		if invite.roomID == roomID {
			invite.expiresAt, invite.userLimit = expiresAt, userLimit
			return key, nil
		}
	}
	s.links[token] = &partnerInvite{roomID: roomID, expiresAt: expiresAt, userLimit: userLimit, users: make(map[int64]bool)}
	return token, nil
}

func (s *partnerInviteStore) current(roomID int64) (string, int64, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, invite := range s.links {
		if invite.roomID == roomID && (invite.expiresAt.IsZero() || invite.expiresAt.After(time.Now())) {
			expiresAt := int64(0)
			if !invite.expiresAt.IsZero() {
				expiresAt = invite.expiresAt.Unix()
			}
			return token, expiresAt, invite.userLimit
		}
	}
	return "", 0, 0
}

func (s *partnerInviteStore) join(token string, roomID, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.joinLocked(token, roomID, userID)
}

func (s *partnerInviteStore) joinLocked(token string, roomID, userID int64) error {
	invite := s.links[token]
	if invite == nil || (!invite.expiresAt.IsZero() && !invite.expiresAt.After(time.Now())) {
		delete(s.links, token)
		return errors.New("partner_invite_expired")
	}
	if roomID != invite.roomID || userID <= 0 {
		return errors.New("signed_room_mismatch")
	}
	if userID == roomID || invite.users[userID] {
		return nil
	}
	if invite.userLimit > 0 && len(invite.users) >= invite.userLimit {
		return errors.New("partner_invite_full")
	}
	invite.users[userID] = true
	return nil
}

func (s *partnerInviteStore) details(roomID int64) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, invite := range s.links {
		if invite.roomID == roomID && (invite.expiresAt.IsZero() || invite.expiresAt.After(time.Now())) {
			return map[string]any{"hasPassword": invite.password != [32]byte{}}
		}
	}
	return map[string]any{"hasPassword": false}
}

// Caller holds the store lock.
func (s *partnerInviteStore) findPassword(roomID int64, value string) string {
	hash := sha256.Sum256([]byte(value))
	for token, invite := range s.links {
		if invite.roomID != roomID {
			continue
		}
		if !invite.expiresAt.IsZero() && !invite.expiresAt.After(time.Now()) {
			continue
		}
		if invite.password != [32]byte{} && hmac.Equal(invite.password[:], hash[:]) {
			return token
		}
	}
	return ""
}

func (s *partnerInviteStore) setPassword(token, password string) error {
	if password != "" && (len(password) < 6 || len(password) > 64 || strings.TrimSpace(password) != password) {
		return errors.New("Use a passcode of 6–64 characters without leading or trailing spaces.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	invite := s.links[token]
	if invite == nil {
		return errors.New("Create an invite first.")
	}
	if password == "" {
		invite.password = [32]byte{}
		return nil
	}
	invite.password = sha256.Sum256([]byte(password))
	return nil
}

func (s *partnerInviteStore) redeem(roomID int64, value string, userID int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID <= 0 {
		return "", errors.New("Telegram authentication required.")
	}
	if !s.allowAttempt(userID) {
		return "", errors.New("Too many attempts. Wait a minute and try again.")
	}
	token := ""
	if roomID > 0 && len(value) <= 64 {
		token = s.findPassword(roomID, strings.TrimSpace(value))
	}
	if token == "" {
		return "", errors.New("Invalid player ID or passcode, or the invite expired. Ask the owner to check them.")
	}
	if err := s.joinLocked(token, roomID, userID); err != nil {
		return "", err
	}
	return token, nil
}

func (s *partnerInviteStore) allowAttempt(userID int64) bool {
	if s.attempts == nil {
		s.attempts = make(map[int64]struct {
			count int
			until time.Time
		})
	}
	now := time.Now()
	for user, attempt := range s.attempts {
		if !attempt.until.After(now) {
			delete(s.attempts, user)
		}
	}
	attempt := s.attempts[userID]
	if attempt.count >= 5 {
		return false
	}
	attempt.count++
	if attempt.until.IsZero() {
		attempt.until = now.Add(time.Minute)
	}
	s.attempts[userID] = attempt
	return true
}
