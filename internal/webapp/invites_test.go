package webapp

import (
	"ashokshau/tg-web/internal/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	td "github.com/AshokShau/gotdbot"
	tgwebapp "github.com/AshokShau/gotdbot/webapp"
)

func setTestBotUser(bot *td.Client, username string) {
	me := &td.User{Usernames: &td.Usernames{EditableUsername: username}}
	field := reflect.ValueOf(bot).Elem().FieldByName("me")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(me))
}

func partnerInitData(userID int64, startParam string) string {
	values := url.Values{"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)}, "start_param": {startParam},
		"user": {`{"id":` + strconv.FormatInt(userID, 10) + `,"first_name":"Partner"}`}}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(config.Token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte("auth_date=" + values.Get("auth_date") + "\nstart_param=" + startParam + "\nuser=" + values.Get("user")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func TestPartnerRequiresSignedLaunch(t *testing.T) {
	useTelegramTestToken(t)
	store := &partnerInviteStore{passwords: make(map[int64][32]byte), admitted: make(map[int64]map[int64]bool), disabledPublic: make(map[int64]bool)}
	access := telegramRoomAccess{invites: store}
	client := voiceTestClient(42, 0, false)
	msg := ClientMessage{RoomID: "42", InitData: partnerInitData(43, "42")}
	if !initializeTelegramSession(client, msg, access) {
		t.Fatal("valid signed public join was rejected")
	}
	p := client.refreshPermissions(nil)
	if !p.CanPlay || !p.CanControl || p.IsAdmin || p.IsAuth {
		t.Fatalf("unexpected partner permissions: %+v", p)
	}
	for _, raw := range []string{
		strings.Replace(msg.InitData, "Partner", "Tampered", 1),
		partnerInitData(43, "50"),
	} {
		if initializeTelegramSession(voiceTestClient(42, 0, false), ClientMessage{RoomID: "42", InitData: raw}, access) {
			t.Fatal("tampered or mismatched join succeeded")
		}
	}
	if !initializeTelegramSession(voiceTestClient(42, 0, false), ClientMessage{RoomID: "42", InitData: partnerInitData(42, "42")}, access) {
		t.Fatal("normal PM owner access regressed")
	}
	if err := validateTelegramRoomAccess(-42, &tgwebapp.WebAppInitData{User: &tgwebapp.WebAppUser{ID: 43}, StartParam: "-42"}, access); err == nil || err.Error() != "group_launch_required" {
		t.Fatalf("group access changed: %v", err)
	}
}

func TestOwnerSharesPlayerLink(t *testing.T) {
	bot := &td.Client{}
	setTestBotUser(bot, "test_bot")
	for _, tc := range []struct{ room, user int64 }{{42, 42}, {42, 43}, {-42, 42}} {
		client := voiceTestClient(tc.room, tc.user, true)
		client.sessionActive = true
		handleClientMessage(bot, client, ClientMessage{Type: "create_partner_invite", RequestID: "invite"})
		var event struct {
			Event, RequestID string
			Data             json.RawMessage
		}
		if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
			t.Fatal(err)
		}
		if tc.room != tc.user || tc.room <= 0 {
			if event.Event != "error" {
				t.Fatal("non-owner created an invite")
			}
			continue
		}
		var data struct{ URL string }
		if err := json.Unmarshal(event.Data, &data); err != nil {
			t.Fatal(err)
		}
		if event.Event != "partner_invite_created" || event.RequestID != "invite" || data.URL != "https://t.me/test_bot/web?startapp=42" {
			t.Fatalf("unexpected invite response: %s", event.Data)
		}
	}
}

func TestOwnerManagesPublicAccess(t *testing.T) {
	previous := partnerInvites
	partnerInvites = &partnerInviteStore{passwords: make(map[int64][32]byte), admitted: make(map[int64]map[int64]bool), disabledPublic: make(map[int64]bool)}
	t.Cleanup(func() { partnerInvites = previous })
	bot := &td.Client{}
	setTestBotUser(bot, "test_bot")

	falseVal := false
	trueVal := true

	// Non-owner cannot change public access
	guest := voiceTestClient(42, 43, true)
	guest.sessionActive = true
	handleClientMessage(bot, guest, ClientMessage{Type: "set_partner_public_access", PublicAccess: &falseVal})
	var errEvent struct{ Event string }
	_ = json.Unmarshal([]byte(<-guest.outbox), &errEvent)
	if errEvent.Event != "error" {
		t.Fatal("non-owner changed public access")
	}

	// Owner disables public access
	owner := voiceTestClient(42, 42, true)
	owner.sessionActive = true
	handleClientMessage(bot, owner, ClientMessage{Type: "set_partner_public_access", PublicAccess: &falseVal})
	var event struct {
		Event string
		Data  struct {
			PublicAccess bool
			URL          string
		}
	}
	if err := json.Unmarshal([]byte(<-owner.outbox), &event); err != nil {
		t.Fatal(err)
	}
	if event.Event != "partner_invite_created" || event.Data.PublicAccess {
		t.Fatalf("unexpected public access response: %+v", event)
	}
	if partnerInvites.isPublicAccess(42) {
		t.Fatal("public access was not disabled")
	}

	// Owner re-enables public access
	handleClientMessage(bot, owner, ClientMessage{Type: "set_partner_public_access", PublicAccess: &trueVal})
	_ = json.Unmarshal([]byte(<-owner.outbox), &event)
	if !event.Data.PublicAccess || !partnerInvites.isPublicAccess(42) {
		t.Fatal("public access was not re-enabled")
	}

	// Setting a passcode automatically marks public access as false
	handleClientMessage(bot, owner, ClientMessage{Type: "set_partner_password", Password: "my-secret-pass"})
	var passEvent struct {
		Event string
		Data  struct {
			HasPassword  bool
			PublicAccess bool
		}
	}
	if err := json.Unmarshal([]byte(<-owner.outbox), &passEvent); err != nil {
		t.Fatal(err)
	}
	if !passEvent.Data.HasPassword || passEvent.Data.PublicAccess {
		t.Fatalf("expected passcode on and public access false, got: %+v", passEvent.Data)
	}
	if partnerInvites.isPublicAccess(42) {
		t.Fatal("public access remained true after setting passcode")
	}

	// Enabling public access while passcode is set removes the passcode
	handleClientMessage(bot, owner, ClientMessage{Type: "set_partner_public_access", PublicAccess: &trueVal})
	_ = json.Unmarshal([]byte(<-owner.outbox), &passEvent)
	if passEvent.Data.HasPassword || !passEvent.Data.PublicAccess {
		t.Fatalf("expected passcode removed and public access true, got: %+v", passEvent.Data)
	}
	if !partnerInvites.isPublicAccess(42) {
		t.Fatal("public access was not restored")
	}
}
