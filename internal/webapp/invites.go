package webapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"strings"
	"sync"
	"time"
)

type partnerInviteStore struct {
	mu             sync.Mutex
	passwords      map[int64][32]byte
	admitted       map[int64]map[int64]bool
	disabledPublic map[int64]bool
	attempts       map[int64]struct {
		count int
		until time.Time
	}
}

var partnerInvites = &partnerInviteStore{
	passwords:      make(map[int64][32]byte),
	admitted:       make(map[int64]map[int64]bool),
	disabledPublic: make(map[int64]bool),
}

func (s *partnerInviteStore) setPassword(roomID int64, password string) error {
	if !validPasscode(password, true) {
		return errors.New("Use a passcode of 4–64 characters without leading or trailing spaces.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.passwords == nil {
		s.passwords = make(map[int64][32]byte)
	}
	if s.disabledPublic == nil {
		s.disabledPublic = make(map[int64]bool)
	}
	if password == "" {
		delete(s.passwords, roomID)
		delete(s.disabledPublic, roomID)
		return nil
	}
	s.passwords[roomID] = sha256.Sum256([]byte(strings.TrimSpace(password)))
	s.disabledPublic[roomID] = true
	return nil
}

func (s *partnerInviteStore) isPublicAccess(roomID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.passwords != nil && s.passwords[roomID] != [32]byte{} {
		return false
	}
	if s.disabledPublic == nil {
		return true
	}
	return !s.disabledPublic[roomID]
}

func (s *partnerInviteStore) setPublicAccess(roomID int64, allow bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disabledPublic == nil {
		s.disabledPublic = make(map[int64]bool)
	}
	if allow {
		delete(s.disabledPublic, roomID)
		if s.passwords != nil {
			delete(s.passwords, roomID)
		}
	} else {
		s.disabledPublic[roomID] = true
	}
}

func (s *partnerInviteStore) details(roomID int64) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	hasPassword := s.passwords != nil && s.passwords[roomID] != [32]byte{}
	return map[string]any{
		"hasPassword":  hasPassword,
		"publicAccess": !hasPassword && (s.disabledPublic == nil || !s.disabledPublic[roomID]),
	}
}

func (s *partnerInviteStore) checkRoomAccess(roomID, userID int64, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if userID <= 0 {
		return errors.New("Telegram authentication required.")
	}
	if userID == roomID {
		return nil
	}
	if s.admitted != nil && s.admitted[roomID] != nil && s.admitted[roomID][userID] {
		return nil
	}

	hasPassword := s.passwords != nil && s.passwords[roomID] != [32]byte{}
	if hasPassword {
		if password == "" {
			return errors.New("partner_password_required")
		}
		if !s.allowAttempt(userID) {
			return errors.New("Too many attempts. Wait a minute and try again.")
		}
		if !validPasscode(password, false) {
			return errors.New("partner_password_invalid")
		}
		hash := sha256.Sum256([]byte(strings.TrimSpace(password)))
		storedHash := s.passwords[roomID]
		if !hmac.Equal(storedHash[:], hash[:]) {
			return errors.New("partner_password_invalid")
		}
		s.admitLocked(roomID, userID)
		return nil
	}

	if s.disabledPublic != nil && s.disabledPublic[roomID] {
		return errors.New("public_access_disabled")
	}

	s.admitLocked(roomID, userID)
	return nil
}

func (s *partnerInviteStore) admitLocked(roomID, userID int64) {
	if s.admitted == nil {
		s.admitted = make(map[int64]map[int64]bool)
	}
	if s.admitted[roomID] == nil {
		s.admitted[roomID] = make(map[int64]bool)
	}
	s.admitted[roomID][userID] = true
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
