package webapp

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func useTelegramTestToken(t *testing.T) {
	t.Helper()
	previous := config.Token
	config.Token = "123456:telegram-auth-test-token"
	t.Cleanup(func() { config.Token = previous })
}

func signedTelegramTestData(values url.Values, token string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	fields := make([]string, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(fields, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func TestTelegramInitDataValidation(t *testing.T) {
	const token = "123456:telegram-auth-test-token"
	for _, tc := range []struct {
		name, reason string
		mutate       func(url.Values)
		changeRaw    func(string) string
	}{
		{name: "signed launch with signature and Unicode"},
		{name: "within future tolerance", mutate: func(v url.Values) { v.Set("auth_date", strconv.FormatInt(time.Now().Add(4*time.Minute).Unix(), 10)) }},
		{name: "inside lifetime", mutate: func(v url.Values) {
			v.Set("auth_date", strconv.FormatInt(time.Now().Add(-24*time.Hour+time.Minute).Unix(), 10))
		}},
		{name: "expired", reason: "expired_auth_date", mutate: func(v url.Values) {
			v.Set("auth_date", strconv.FormatInt(time.Now().Add(-24*time.Hour-time.Minute).Unix(), 10))
		}},
		{name: "future", reason: "future_auth_date", mutate: func(v url.Values) { v.Set("auth_date", strconv.FormatInt(time.Now().Add(6*time.Minute).Unix(), 10)) }},
		{name: "missing date", reason: "invalid_auth_date", mutate: func(v url.Values) { v.Del("auth_date") }},
		{name: "partial date", reason: "invalid_auth_date", mutate: func(v url.Values) { v.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10)+"oops") }},
		{name: "missing user", reason: "invalid_user", mutate: func(v url.Values) { v.Del("user") }},
		{name: "null user", reason: "invalid_user", mutate: func(v url.Values) { v.Set("user", "null") }},
		{name: "invalid user ID", reason: "invalid_user", mutate: func(v url.Values) { v.Set("user", `{"id":0}`) }},
		{name: "malformed user", reason: "malformed_init_data", mutate: func(v url.Values) { v.Set("user", "{") }},
		{name: "duplicate user", reason: "malformed_init_data", changeRaw: func(raw string) string { return raw + "&user=" + url.QueryEscape(`{"id":99}`) }},
		{name: "duplicate hash", reason: "malformed_init_data", changeRaw: func(raw string) string { return raw + "&hash=other" }},
		{name: "tampered user", reason: "hash_mismatch", changeRaw: func(raw string) string { return strings.Replace(raw, "Morgan", "Forged", 1) }},
		{name: "tampered signature", reason: "hash_mismatch", changeRaw: func(raw string) string { return strings.Replace(raw, "test-signature", "forged-signature", 1) }},
		{name: "malformed query", reason: "malformed_init_data", changeRaw: func(raw string) string { return raw + "&bad=%xx" }},
		{name: "empty", reason: "missing_init_data", changeRaw: func(string) string { return "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{
				"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)},
				"user":      {`{"id":42,"first_name":"Alex + é","last_name":"Morgan","allows_write_to_pm":true}`},
				"signature": {"test-signature"}, "chat_type": {"supergroup"}, "chat_instance": {"signed-instance"}, "start_param": {"-42"},
			}
			if tc.mutate != nil {
				tc.mutate(values)
			}
			raw := signedTelegramTestData(values, token)
			if tc.changeRaw != nil {
				raw = tc.changeRaw(raw)
			}
			data, err := validateTelegramInitData(raw, token)
			if tc.reason != "" {
				if err == nil || err.Error() != tc.reason || data != nil {
					t.Fatalf("expected %s, got data=%v error=%v", tc.reason, data, err)
				}
				return
			}
			if err != nil || data == nil || data.User == nil || data.User.ID != 42 || !data.User.AllowsWriteToPM || data.User.FirstName != "Alex + é" {
				t.Fatalf("valid signed user was lost: data=%v error=%v", data, err)
			}
			context := telegramContext(data)
			if !context.AuthValid || context.ChatType != "supergroup" || context.ChatInstance != "signed-instance" || context.StartParam != "-42" {
				t.Fatalf("signed room context was lost: %+v", context)
			}
		})
	}
	if _, err := validateTelegramInitData("auth_date=1", ""); err == nil || err.Error() != "missing_bot_token" {
		t.Fatalf("missing bot token accepted: %v", err)
	}
}

func TestPlaybackModesDoNotGrantSettingsAuthority(t *testing.T) {
	for _, tc := range []struct {
		name                string
		admin, auth         bool
		mode                string
		restrictedPlay      bool
		canControl, canPlay bool
	}{
		{"admin", true, false, utils.Admins, true, true, true},
		{"authorized", false, true, utils.Admins, true, true, true},
		{"everyone", false, false, utils.Everyone, false, true, true},
		{"everyone controls but restricted play", false, false, utils.Everyone, true, true, false},
		{"listener can request", false, false, utils.Admins, false, false, true},
		{"restricted listener", false, false, utils.Admins, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := permissionsForRole(tc.admin, tc.auth, tc.mode, tc.restrictedPlay)
			if p.IsAdmin != tc.admin || p.IsAuth != tc.auth || p.CanControl != tc.canControl || p.CanPlay != tc.canPlay {
				t.Fatalf("unexpected permissions: %+v", p)
			}
			client := voiceTestClient(-99904, 20, false)
			client.SetPermissions(p)
			client.sendUserInfo()
			var event struct {
				Data struct{ IsAdmin, IsAuth, CanControl, CanPlay bool }
			}
			if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
				t.Fatal(err)
			}
			if event.Data.IsAdmin != tc.admin || event.Data.IsAuth != tc.auth || event.Data.CanControl != tc.canControl || event.Data.CanPlay != tc.canPlay {
				t.Fatalf("role changed in user_info: %+v", event.Data)
			}
		})
	}
}

func TestDevelopersReceiveAdminAuthority(t *testing.T) {
	previous := config.DEVS
	config.DEVS = []int64{901}
	t.Cleanup(func() { config.DEVS = previous })
	p := resolveRoomPermissions(nil, -99905, 901)
	if !p.IsAdmin || !p.CanControl || !p.CanPlay {
		t.Fatalf("unexpected bot admin: %+v", p)
	}
	p = resolveRoomPermissions(nil, 902, 902)
	if !p.IsAdmin || !p.CanControl || !p.CanPlay {
		t.Fatalf("unexpected private room owner: %+v", p)
	}
	if p := resolveRoomPermissions(nil, 903, 901); p != (roomPermissions{}) {
		t.Fatalf("another private room granted permissions: %+v", p)
	}
}

func TestNonAdminsCannotChangeRoomSettings(t *testing.T) {
	for _, auth := range []bool{false, true} {
		for _, command := range []string{"loop", "autoplay", "chat_settings", "vc_admin_set_join_muted"} {
			t.Run(command, func(t *testing.T) {
				client := voiceTestClient(-99906, 21, false)
				client.sessionActive = true // Exercise settings authority in an admitted session.
				client.IsAuth, client.CanControl, client.CanPlay, client.AllowsWriteToPM = auth, true, true, true
				value := true
				handleClientMessage(nil, client, ClientMessage{Type: command, RequestID: "settings", ChatEnabled: &value, Muted: true})
				var event struct {
					Event, RequestID, Command string
					Data                      string
				}
				if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
					t.Fatal(err)
				}
				if event.Event != "error" || !strings.Contains(event.Data, "Admin permission") || event.RequestID != "settings" || event.Command != command {
					t.Fatalf("non-admin settings command was not denied: %+v", event)
				}
			})
		}
	}
}
