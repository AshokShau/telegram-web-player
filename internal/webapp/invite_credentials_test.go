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
	if err := store.setPassword(token, "1234"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.redeem(42, "1234", 43); err != nil || got != token {
		t.Fatalf("PIN join failed: %v", err)
	}
	if _, err := store.redeem(42, "1234", 44); err == nil || err.Error() != "partner_invite_full" {
		t.Fatalf("password bypassed user limit: %v", err)
	}
	if _, err := store.redeem(42, "1234", 43); err != nil {
		t.Fatalf("repeat password join failed: %v", err)
	}
	if err := store.join(token, 42, 43); err != nil {
		t.Fatal("repeat link join failed")
	}
	if err := store.join(token, 42, 44); err == nil {
		t.Fatal("link bypassed password user limit")
	}
	encoded, _ := json.Marshal(store.details(42))
	if strings.Contains(string(encoded), "1234") || store.links[token].password == [32]byte{} {
		t.Fatal("password was not kept private as a hash")
	}
	if err := store.setPassword(token, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.redeem(42, "1234", 43); err == nil {
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
	for _, password := range []string{"123", " 123456", strings.Repeat("x", 65)} {
		if store.setPassword(token, password) == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if err := store.setPassword(token, "shared-password"); err != nil {
		t.Fatal(err)
	}
	if err := store.setPassword(other, "shared-password"); err != nil {
		t.Fatal("another owner could not use the same password")
	}
	for range 5 {
		if _, err := store.redeem(42, "incorrect", 43); err == nil {
			t.Fatal("incorrect password accepted")
		}
	}
	if _, err := store.redeem(42, "shared-password", 43); err == nil || !strings.Contains(err.Error(), "Too many attempts") {
		t.Fatal("guess limit not enforced")
	}
	attempt := store.attempts[43]
	attempt.until = time.Now().Add(-time.Second)
	store.attempts[43] = attempt
	if _, err := store.redeem(42, "shared-password", 43); err != nil {
		t.Fatal("guess limit did not reset")
	}
	store.links[token].expiresAt = time.Now().Add(-time.Second)
	if _, err := store.redeem(42, "shared-password", 43); err == nil {
		t.Fatal("expired password accepted")
	}
	restarted := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	if _, err := restarted.redeem(42, "shared-password", 43); err == nil {
		t.Fatal("password survived restart")
	}
	if _, err := store.redeem(42, "shared-password", 0); err == nil {
		t.Fatal("anonymous password join accepted")
	}
}

func TestPartnerPasswordsAreScopedToPlayer(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	first, _ := store.create(42, time.Time{}, 1)
	second, _ := store.create(50, time.Time{}, 1)
	for _, token := range []string{first, second} {
		if err := store.setPassword(token, "same-password"); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		room  int64
		token string
	}{{42, first}, {50, second}} {
		if token, err := store.redeem(tc.room, "same-password", 43); err != nil || token != tc.token {
			t.Fatalf("password joined the wrong player: %v", err)
		}
		if _, err := store.redeem(tc.room, "same-password", 44); err == nil || err.Error() != "partner_invite_full" {
			t.Fatalf("player user limit was not respected: %v", err)
		}
	}
	if err := store.setPassword(first, "new-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.redeem(42, "same-password", 43); err == nil {
		t.Fatal("another player's password granted access")
	}
	if token, err := store.redeem(50, "same-password", 43); err != nil || token != second {
		t.Fatal("changing one password affected another player")
	}
	if _, err := store.redeem(99, "same-password", 43); err == nil {
		t.Fatal("unknown player ID was accepted")
	}
}

func TestPartnerSessionRequiresTelegramAuthentication(t *testing.T) {
	useTelegramTestToken(t)
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

func TestPlayerLinkRequiresPasswordBeforeAdmission(t *testing.T) {
	useTelegramTestToken(t)
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, _ := store.create(42, time.Time{}, 1)
	if err := store.setPassword(token, "shared-password"); err != nil {
		t.Fatal(err)
	}
	access := telegramRoomAccess{invites: store}
	client := voiceTestClient(42, 0, false)
	msg := ClientMessage{RoomID: "42", InitData: partnerInitData(43, "42")}
	if initializeTelegramSession(client, msg, access) {
		t.Fatal("player link admitted an uninvited user")
	}
	var challenge struct{ Code string }
	var event struct{ Data map[string]any }
	if err := json.Unmarshal([]byte(<-client.outbox), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Code != "partner_password_required" || client.UserID != 0 || len(store.links[token].users) != 0 {
		t.Fatal("password challenge granted access or consumed a spot")
	}
	msg.Password = "incorrect"
	if initializeTelegramSession(client, msg, access) {
		t.Fatal("incorrect password admitted a user")
	}
	var rejection struct{ Code string }
	if err := json.Unmarshal([]byte(<-client.outbox), &rejection); err != nil {
		t.Fatal(err)
	}
	if rejection.Code != "partner_password_invalid" {
		t.Fatal("password retry was not offered")
	}
	msg.Password = "shared-password"
	if !initializeTelegramSession(client, msg, access) {
		t.Fatal("correct password rejected")
	}
	client.sendUserInfo()
	if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
		t.Fatal(err)
	}
	if event.Data["partnerInviteToken"] != token || len(store.links[token].users) != 1 {
		t.Fatal("joined session did not receive its access proof")
	}
	reconnect := ClientMessage{RoomID: "42", InitData: msg.InitData, PartnerInvite: token}
	if !initializeTelegramSession(voiceTestClient(42, 0, false), reconnect, access) || len(store.links[token].users) != 1 {
		t.Fatal("reconnect required password again or consumed another spot")
	}
	msg.InitData = partnerInitData(44, "42")
	if initializeTelegramSession(voiceTestClient(42, 0, false), msg, access) {
		t.Fatal("password bypassed distinct-user limit")
	}
	msg.InitData = partnerInitData(43, "50")
	if initializeTelegramSession(voiceTestClient(42, 0, false), msg, access) {
		t.Fatal("password bypassed signed room mismatch")
	}
	msg.InitData = partnerInitData(42, "42")
	msg.Password = ""
	if !initializeTelegramSession(voiceTestClient(42, 0, false), msg, access) {
		t.Fatal("owner was asked for a password")
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
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
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
	if initializeTelegramSession(clientDisabled, msg, access) {
		t.Fatal("disabled public access admitted an ordinary join")
	}
	var challenge struct{ Code string }
	if err := json.Unmarshal([]byte(<-clientDisabled.outbox), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Code != "public_access_disabled" {
		t.Fatalf("expected public_access_disabled, got %s", challenge.Code)
	}

	// Valid invite token still works when public access is disabled
	token, err := store.create(42, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	inviteClient := voiceTestClient(42, 0, false)
	inviteMsg := ClientMessage{RoomID: "42", InitData: partnerInitData(43, token)}
	if !initializeTelegramSession(inviteClient, inviteMsg, access) {
		t.Fatal("valid invite rejected when public access disabled")
	}

	// Passcode works when public access is disabled
	if err := store.setPassword(token, "test-pass"); err != nil {
		t.Fatal(err)
	}
	passClient := voiceTestClient(42, 0, false)
	if initializeTelegramSession(passClient, msg, access) {
		t.Fatal("unauthenticated join succeeded on passcode-protected room")
	}
	if err := json.Unmarshal([]byte(<-passClient.outbox), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Code != "partner_password_required" {
		t.Fatalf("expected partner_password_required, got %s", challenge.Code)
	}
	msg.Password = "test-pass"
	if !initializeTelegramSession(voiceTestClient(42, 0, false), msg, access) {
		t.Fatal("correct passcode rejected when public access disabled")
	}

	// Public access enabled cannot bypass user limit
	store.setPublicAccess(42, true)
	limitStore := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	limToken, _ := limitStore.create(42, time.Time{}, 1)
	limAccess := telegramRoomAccess{invites: limitStore}
	firstClient := voiceTestClient(42, 0, false)
	if !initializeTelegramSession(firstClient, ClientMessage{RoomID: "42", InitData: partnerInitData(43, "42")}, limAccess) {
		t.Fatal("first public user rejected")
	}
	secondClient := voiceTestClient(42, 0, false)
	if initializeTelegramSession(secondClient, ClientMessage{RoomID: "42", InitData: partnerInitData(44, "42")}, limAccess) {
		t.Fatal("public join bypassed user limit")
	}
	var fullErr struct{ Code string }
	if err := json.Unmarshal([]byte(<-secondClient.outbox), &fullErr); err != nil {
		t.Fatal(err)
	}
	if fullErr.Code != "partner_invite_full" {
		t.Fatalf("expected partner_invite_full, got %s", fullErr.Code)
	}
	_ = limToken

	// Public access enabled cannot bypass expiry
	expStore := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	expToken, err := expStore.create(42, time.Now().Add(time.Hour), 0)
	if err != nil {
		t.Fatal(err)
	}
	expStore.links[expToken].expiresAt = time.Now().Add(-time.Second)
	expAccess := telegramRoomAccess{invites: expStore}
	expClient := voiceTestClient(42, 0, false)
	if initializeTelegramSession(expClient, ClientMessage{RoomID: "42", InitData: partnerInitData(43, "42")}, expAccess) {
		t.Fatal("public join bypassed invite expiry")
	}
	var expErr struct{ Code string }
	if err := json.Unmarshal([]byte(<-expClient.outbox), &expErr); err != nil {
		t.Fatal(err)
	}
	if expErr.Code != "partner_invite_expired" {
		t.Fatalf("expected partner_invite_expired, got %s", expErr.Code)
	}
}
