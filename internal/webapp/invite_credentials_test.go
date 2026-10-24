package webapp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPartnerPasswordAndLinksShareLimits(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, err := store.create(42, time.Time{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.setPassword(token, "123456"); err != nil {
		t.Fatal(err)
	}
	if got, room, err := store.redeem("123456", 43); err != nil || got != token || room != 42 {
		t.Fatalf("PIN join failed: %v", err)
	}
	if _, _, err := store.redeem("123456", 44); err == nil || err.Error() != "partner_invite_full" {
		t.Fatalf("password bypassed user limit: %v", err)
	}
	if _, _, err := store.redeem("123456", 43); err != nil {
		t.Fatalf("repeat password join failed: %v", err)
	}
	if err := store.join(token, 42, 43); err != nil {
		t.Fatal("repeat link join failed")
	}
	if err := store.join(token, 42, 44); err == nil {
		t.Fatal("link bypassed password user limit")
	}
	encoded, _ := json.Marshal(store.details(42))
	if strings.Contains(string(encoded), "123456") || store.links[token].password == [32]byte{} {
		t.Fatal("password was not kept private as a hash")
	}
	if err := store.setPassword(token, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.redeem("123456", 43); err == nil {
		t.Fatal("removed password still grants access")
	}
	if err := store.join(token, 42, 43); err != nil {
		t.Fatal("removing the password disabled link access")
	}
}

func TestPartnerCredentialsValidationExpiryAndRateLimit(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, _ := store.create(42, time.Now().Add(time.Hour), 0)
	other, _ := store.create(50, time.Time{}, 0)
	for _, password := range []string{"12345", " 123456", strings.Repeat("x", 65)} {
		if store.setPassword(token, password) == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if err := store.setPassword(token, "shared-password"); err != nil {
		t.Fatal(err)
	}
	if store.setPassword(other, "shared-password") == nil {
		t.Fatal("ambiguous password accepted")
	}
	for range 5 {
		if _, _, err := store.redeem("incorrect", 43); err == nil {
			t.Fatal("incorrect password accepted")
		}
	}
	if _, _, err := store.redeem("shared-password", 43); err == nil || !strings.Contains(err.Error(), "Too many attempts") {
		t.Fatal("guess limit not enforced")
	}
	attempt := store.attempts[43]
	attempt.until = time.Now().Add(-time.Second)
	store.attempts[43] = attempt
	if _, _, err := store.redeem("shared-password", 43); err != nil {
		t.Fatal("guess limit did not reset")
	}
	store.links[token].expiresAt = time.Now().Add(-time.Second)
	if _, _, err := store.redeem("shared-password", 43); err == nil {
		t.Fatal("expired password accepted")
	}
	restarted := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	if _, _, err := restarted.redeem("shared-password", 43); err == nil {
		t.Fatal("password survived restart")
	}
	if _, _, err := store.redeem("shared-password", 0); err == nil {
		t.Fatal("anonymous password join accepted")
	}
}

func TestPartnerSessionRequiresTelegramAuthentication(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, _ := store.create(42, time.Time{}, 0)
	access := telegramRoomAccess{invites: store}
	msg := ClientMessage{RoomID: "42", InitData: partnerInitData(43, "43"), PartnerInvite: token}
	client := voiceTestClient(42, 0, false)
	if !initializeTelegramSession(client, msg, access) {
		t.Fatal("partner proof with valid Telegram identity was rejected")
	}
	permissions := client.refreshPermissions(nil)
	if !permissions.CanPlay || !permissions.CanControl || permissions.IsAdmin {
		t.Fatal("incorrect partner permissions")
	}
	for _, invalid := range []ClientMessage{
		{RoomID: "42", InitData: msg.InitData, PartnerInvite: "unknown"},
		{RoomID: "42", InitData: strings.Replace(msg.InitData, "Partner", "Forged", 1), PartnerInvite: token},
		{RoomID: "42", InitData: msg.InitData},
	} {
		if initializeTelegramSession(voiceTestClient(42, 0, false), invalid, access) {
			t.Fatal("unverified partner session admitted")
		}
	}
	msg.RoomID = "50"
	if initializeTelegramSession(voiceTestClient(50, 0, false), msg, access) {
		t.Fatal("partner proof accessed another room")
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
