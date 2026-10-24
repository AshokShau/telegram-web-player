package webapp

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"sync"
	"time"
)

type partnerInvite struct {
	roomID    int64
	expiresAt time.Time
	userLimit int
	users     map[int64]bool
}

type partnerInviteStore struct {
	mu    sync.Mutex
	links map[string]*partnerInvite
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
