package webapp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPartnerCredentialsValidationAndRateLimit(t *testing.T) {
	store := &partnerInviteStore{passwords: make(map[int64][32]byte), admitted: make(map[int64]map[int64]bool), disabledPublic: make(map[int64]bool)}
	for _, password := range []string{"123", " 123456", strings.Repeat("x", 65)} {
		if store.setPassword(42, password) == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if err := store.setPassword(42, "shared-password"); err != nil {
		t.Fatal(err)
	}
	if err := store.setPassword(50, "shared-password"); err != nil {
		t.Fatal("another owner could not use the same password")
	}
	for range 5 {
		if err := store.checkRoomAccess(42, 43, "incorrect"); err == nil {
			t.Fatal("incorrect password accepted")
		}
	}
	if err := store.checkRoomAccess(42, 43, "shared-password"); err == nil || !strings.Contains(err.Error(), "Too many attempts") {
		t.Fatal("guess limit not enforced")
	}
	attempt := store.attempts[43]
	attempt.until = time.Now().Add(-time.Second)
	store.attempts[43] = attempt
	if err := store.checkRoomAccess(42, 43, "shared-password"); err != nil {
		t.Fatal("guess limit did not reset")
	}
	restarted := &partnerInviteStore{passwords: make(map[int64][32]byte), admitted: make(map[int64]map[int64]bool), disabledPublic: make(map[int64]bool)}
	if err := restarted.checkRoomAccess(42, 43, "shared-password"); err != nil {
		t.Fatal("restarted store without password should be public by default")
	}
	if err := store.checkRoomAccess(42, 0, "shared-password"); err == nil {
		t.Fatal("anonymous password join accepted")
	}
}

func TestPartnerPasswordsAreScopedToPlayer(t *testing.T) {
	store := &partnerInviteStore{passwords: make(map[int64][32]byte), admitted: make(map[int64]map[int64]bool), disabledPublic: make(map[int64]bool)}
	for _, roomID := range []int64{42, 50} {
		if err := store.setPassword(roomID, "pass-"+string(rune('A'+int(roomID%10)))); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.checkRoomAccess(42, 43, "pass-B"); err == nil {
		t.Fatal("room 50 password admitted user to room 42")
	}
	if err := store.checkRoomAccess(42, 43, "pass-C"); err != nil {
		t.Fatal("room 42 password failed to admit user")
	}
	// Once admitted, repeat checks without password succeed
	if err := store.checkRoomAccess(42, 43, ""); err != nil {
		t.Fatal("previously admitted user was prompted again")
	}
}

func TestWritePermissionWarningAndRecovery(t *testing.T) {
	client := voiceTestClient(42, 42, true)
	client.sessionActive, client.CanControl = true, true
	handleClientMessage(nil, client, ClientMessage{Type: "pause"})
	var event struct{ Event, Code string }
	if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
		t.Fatal(err)
	}
	if event.Event != "error" || event.Code != "write_access_required" {
		t.Fatal("missing actionable write permission error")
	}
	handleClientMessage(nil, client, ClientMessage{Type: "write_access_granted"})
	_, _, _, _, allowed, _ := client.GetInfo()
	if !allowed {
		t.Fatal("permission grant did not unlock controls")
	}
}

func TestPublicJoinAndAccessModes(t *testing.T) {
	useTelegramTestToken(t)
	store := &partnerInviteStore{passwords: make(map[int64][32]byte), admitted: make(map[int64]map[int64]bool), disabledPublic: make(map[int64]bool)}
	access := telegramRoomAccess{invites: store}

	// Default public join
	client := voiceTestClient(42, 0, false)
	msg := ClientMessage{RoomID: "42", InitData: partnerInitData(43, "42")}
	if !initializeTelegramSession(client, msg, access) {
		t.Fatal("default public join was rejected")
	}
	permissions := client.refreshPermissions(nil)
	if !permissions.CanPlay || !permissions.CanControl || permissions.IsAdmin {
		t.Fatalf("unexpected public partner permissions: %+v", permissions)
	}

	// Disabled public access without passcode rejects player link
	store.setPublicAccess(42, false)
	clientDisabled := voiceTestClient(42, 0, false)
	disabledMsg := ClientMessage{RoomID: "42", InitData: partnerInitData(44, "42")}
	if initializeTelegramSession(clientDisabled, disabledMsg, access) {
		t.Fatal("disabled public access admitted an unadmitted join")
	}
	var challenge struct{ Code string }
	if err := json.Unmarshal([]byte(<-clientDisabled.outbox), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Code != "public_access_disabled" {
		t.Fatalf("expected public_access_disabled, got %s", challenge.Code)
	}

	// Passcode protection requires passcode
	if err := store.setPassword(42, "test-pass"); err != nil {
		t.Fatal(err)
	}
	passClient := voiceTestClient(42, 0, false)
	passMsg := ClientMessage{RoomID: "42", InitData: partnerInitData(45, "42")}
	if initializeTelegramSession(passClient, passMsg, access) {
		t.Fatal("unauthenticated join succeeded on passcode-protected room")
	}
	if err := json.Unmarshal([]byte(<-passClient.outbox), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Code != "partner_password_required" {
		t.Fatalf("expected partner_password_required, got %s", challenge.Code)
	}

	// Wrong passcode is rejected
	passMsg.Password = "wrong-pass"
	wrongClient := voiceTestClient(42, 0, false)
	if initializeTelegramSession(wrongClient, passMsg, access) {
		t.Fatal("incorrect passcode admitted user")
	}
	if err := json.Unmarshal([]byte(<-wrongClient.outbox), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Code != "partner_password_invalid" {
		t.Fatalf("expected partner_password_invalid, got %s", challenge.Code)
	}

	// Correct passcode succeeds
	passMsg.Password = "test-pass"
	correctClient := voiceTestClient(42, 0, false)
	if !initializeTelegramSession(correctClient, passMsg, access) {
		t.Fatal("correct passcode rejected")
	}

	// Owner can always join without passcode
	ownerClient := voiceTestClient(42, 0, false)
	ownerMsg := ClientMessage{RoomID: "42", InitData: partnerInitData(42, "42")}
	if !initializeTelegramSession(ownerClient, ownerMsg, access) {
		t.Fatal("owner was rejected from own room")
	}
}
