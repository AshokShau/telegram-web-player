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
	"sync"
	"sync/atomic"
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

func TestPartnerInviteLimitAndLifetime(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, err := store.create(42, time.Now().Add(time.Hour), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []int64{42, 43, 43, 42} {
		if err := store.join(token, 42, user); err != nil {
			t.Fatalf("repeat or owner join: %v", err)
		}
	}
	if err := store.join(token, 42, 44); err == nil || err.Error() != "partner_invite_full" {
		t.Fatalf("new user bypassed limit: %v", err)
	}
	if err := store.join(token, 44, 43); err == nil {
		t.Fatal("invite admitted a user to another room")
	}
	if err := store.join(token, -42, 43); err == nil {
		t.Fatal("PM invite admitted a group join")
	}
	restarted := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	if err := restarted.join(token, 42, 43); err == nil {
		t.Fatal("invite survived restart")
	}
	store.links[token].expiresAt = time.Now().Add(-time.Second)
	if err := store.join(token, 42, 43); err == nil || err.Error() != "partner_invite_expired" {
		t.Fatalf("returning user bypassed expiry: %v", err)
	}
	if len(store.links) != 0 {
		t.Fatal("expired invite was retained")
	}
	for _, tc := range []struct {
		room, expiry int64
		limit        int
	}{{42, time.Now().Unix() - 1, 1}, {42, time.Now().Unix() + 60, -1}, {-42, time.Now().Unix() + 60, 1}} {
		if _, err := store.create(tc.room, time.Unix(tc.expiry, 0), tc.limit); err == nil {
			t.Fatal("invalid invite settings were accepted")
		}
	}
}

func TestPartnerInviteUnlimitedAndSettingsUpdates(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, err := store.create(42, time.Unix(0, 0), 0)
	if err != nil {
		t.Fatal(err)
	}
	for user := int64(100); user < 200; user++ {
		if err := store.join(token, 42, user); err != nil {
			t.Fatalf("unlimited invite rejected user %d: %v", user, err)
		}
	}
	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	updated, err := store.create(42, expiry, 1)
	if err != nil || updated != token {
		t.Fatalf("updating settings replaced the link: %v", err)
	}
	if err := store.join(token, 42, 100); err != nil {
		t.Fatal("updating settings lost admitted users")
	}
	if err := store.join(token, 42, 200); err == nil {
		t.Fatal("updated limit did not block a new user")
	}
	if current, expiresAt, limit := store.current(42); current != token || expiresAt != expiry.Unix() || limit != 1 {
		t.Fatal("current settings do not match the saved settings")
	}
	updated, err = store.create(42, time.Unix(0, 0), 0)
	if err != nil || updated != token || store.join(token, 42, 200) != nil {
		t.Fatal("could not remove limits from the current link")
	}
	if _, expiresAt, limit := store.current(42); expiresAt != 0 || limit != 0 {
		t.Fatal("unlimited settings were not returned as zero")
	}
	restarted := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	if restarted.join(token, 42, 100) == nil {
		t.Fatal("unlimited invite survived restart")
	}
}

func TestPartnerInviteConcurrentJoins(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, err := store.create(42, time.Now().Add(time.Hour), 3)
	if err != nil {
		t.Fatal(err)
	}
	var joined atomic.Int32
	var group sync.WaitGroup
	for user := int64(100); user < 132; user++ {
		group.Go(func() {
			if store.join(token, 42, user) == nil {
				joined.Add(1)
			}
		})
	}
	group.Wait()
	if joined.Load() != 3 {
		t.Fatalf("concurrent joins admitted %d users", joined.Load())
	}
}

func partnerInitData(userID int64, token string) string {
	values := url.Values{"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)}, "start_param": {token},
		"user": {`{"id":` + strconv.FormatInt(userID, 10) + `,"first_name":"Partner"}`}}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(config.Token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte("auth_date=" + values.Get("auth_date") + "\nstart_param=" + token + "\nuser=" + values.Get("user")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func TestPartnerInviteRequiresSignedLaunch(t *testing.T) {
	useTelegramTestToken(t)
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	token, err := store.create(42, time.Now().Add(time.Hour), 1)
	if err != nil {
		t.Fatal(err)
	}
	access := telegramRoomAccess{invites: store}
	client := voiceTestClient(42, 0, false)
	msg := ClientMessage{RoomID: "42", InitData: partnerInitData(43, token)}
	if !initializeTelegramSession(client, msg, access) {
		t.Fatal("valid signed invite was rejected")
	}
	p := client.refreshPermissions(nil)
	if !p.CanPlay || !p.CanControl || p.IsAdmin || p.IsAuth {
		t.Fatalf("unexpected partner permissions: %+v", p)
	}
	for _, raw := range []string{
		strings.Replace(msg.InitData, "Partner", "Tampered", 1),
		partnerInitData(44, "42"),
		partnerInitData(44, token),
	} {
		if initializeTelegramSession(voiceTestClient(42, 0, false), ClientMessage{RoomID: "42", InitData: raw}, access) {
			t.Fatal("tampered, uninvited or over-limit join succeeded")
		}
	}
	if !initializeTelegramSession(voiceTestClient(42, 0, false), ClientMessage{RoomID: "42", InitData: partnerInitData(42, "42")}, access) {
		t.Fatal("normal PM owner access regressed")
	}
	if err := validateTelegramRoomAccess(-42, &tgwebapp.WebAppInitData{User: &tgwebapp.WebAppUser{ID: 43}, StartParam: "-42"}, access); err == nil || err.Error() != "group_launch_required" {
		t.Fatalf("group access changed: %v", err)
	}
}

func TestOnlyPMOwnerCreatesPartnerInvites(t *testing.T) {
	bot := &td.Client{}
	setTestBotUser(bot, "test_bot")
	for _, tc := range []struct{ room, user int64 }{{42, 42}, {42, 43}, {-42, 42}} {
		client := voiceTestClient(tc.room, tc.user, true)
		client.sessionActive = true
		handleClientMessage(bot, client, ClientMessage{Type: "create_partner_invite", RequestID: "invite", ExpiresAt: time.Now().Add(time.Hour).Unix(), UserLimit: 1})
		var event struct {
			Event, RequestID string
			Data             json.RawMessage
		}
		if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
			t.Fatal(err)
		}
		if tc.room != tc.user {
			if event.Event != "error" {
				t.Fatal("non-owner created an invite")
			}
			continue
		}
		var data struct{ URL string }
		if err := json.Unmarshal(event.Data, &data); err != nil {
			t.Fatal(err)
		}
		if event.Event != "partner_invite_created" || event.RequestID != "invite" || !strings.HasPrefix(data.URL, "https://t.me/test_bot/web?startapp=pm_42_") {
			t.Fatalf("unexpected invite response: %s", event.Data)
		}
	}
}

func TestOwnerManagesCurrentPartnerInvite(t *testing.T) {
	previous := partnerInvites
	partnerInvites = &partnerInviteStore{links: make(map[string]*partnerInvite)}
	t.Cleanup(func() { partnerInvites = previous })
	bot := &td.Client{}
	setTestBotUser(bot, "test_bot")
	expiry := time.Now().Add(time.Hour).Unix()
	var sharedURL string
	for _, command := range []ClientMessage{
		{Type: "create_partner_invite"},
		{Type: "update_partner_invite", ExpiresAt: expiry, UserLimit: 1},
		{Type: "create_partner_invite"},
		{Type: "update_partner_invite"},
	} {
		client := voiceTestClient(42, 42, true)
		client.sessionActive = true
		handleClientMessage(bot, client, command)
		var event struct {
			Event string
			Data  struct {
				URL       string
				ExpiresAt int64
				UserLimit int
			}
		}
		if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
			t.Fatal(err)
		}
		if event.Event != "partner_invite_created" || event.Data.URL == "" {
			t.Fatal("invite request failed")
		}
		if sharedURL == "" {
			sharedURL = event.Data.URL
		}
		if event.Data.URL != sharedURL {
			t.Fatal("saving settings or sharing replaced the invite")
		}
		_, savedExpiry, savedLimit := partnerInvites.current(42)
		if event.Data.ExpiresAt != savedExpiry || event.Data.UserLimit != savedLimit {
			t.Fatal("response does not reflect the current invite settings")
		}
		if command.Type == "update_partner_invite" && (savedExpiry != command.ExpiresAt || savedLimit != command.UserLimit) {
			t.Fatal("settings update was not saved")
		}
	}
}
