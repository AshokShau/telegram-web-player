package webapp

import (
	"ashokshau/tg-web/internal/config"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestClientMessageInputBoundaries(t *testing.T) {
	previous := config.SongDurationLimit
	config.SongDurationLimit = 300
	t.Cleanup(func() { config.SongDurationLimit = previous })
	for _, tc := range []struct {
		name  string
		msg   ClientMessage
		valid bool
	}{
		{"search exact", ClientMessage{Type: "search", Query: strings.Repeat("界", 200)}, true},
		{"search too long", ClientMessage{Type: "search", Query: strings.Repeat("界", 201)}, false},
		{"playlist exact", ClientMessage{Type: "create_playlist", PlaylistName: strings.Repeat("界", 40)}, true},
		{"playlist too long", ClientMessage{Type: "create_playlist", PlaylistName: strings.Repeat("界", 41)}, false},
		{"playlist whitespace", ClientMessage{Type: "rename_playlist", PlaylistName: "   "}, false},
		{"chat exact", ClientMessage{Type: "chat_message", Text: strings.Repeat("😀", 500)}, true},
		{"chat too long", ClientMessage{Type: "chat_message", Text: strings.Repeat("😀", 501)}, false},
		{"passcode minimum", ClientMessage{Type: "set_partner_password", Password: "1234"}, true},
		{"passcode unicode minimum", ClientMessage{Type: "set_partner_password", Password: "音楽共有"}, true},
		{"passcode exact", ClientMessage{Type: "set_partner_password", Password: strings.Repeat("界", 64)}, true},
		{"passcode too long", ClientMessage{Type: "set_partner_password", Password: strings.Repeat("界", 65)}, false},
		{"passcode too short", ClientMessage{Type: "set_partner_password", Password: "abc"}, false},
		{"passcode spaces", ClientMessage{Type: "set_partner_password", Password: " secret"}, false},
		{"remove passcode", ClientMessage{Type: "set_partner_password"}, true},
		{"invite unlimited", ClientMessage{Type: "update_partner_invite"}, true},
		{"invite maximum", ClientMessage{Type: "update_partner_invite", UserLimit: 1000}, true},
		{"invite above maximum", ClientMessage{Type: "update_partner_invite", UserLimit: 1001}, false},
		{"invite huge", ClientMessage{Type: "update_partner_invite", UserLimit: math.MaxInt}, false},
		{"invite negative", ClientMessage{Type: "update_partner_invite", UserLimit: -1}, false},
		{"expiry future", ClientMessage{Type: "update_partner_invite", ExpiresAt: time.Now().Add(time.Hour).Unix()}, true},
		{"expiry last date", ClientMessage{Type: "update_partner_invite", ExpiresAt: maxInviteExpiry}, true},
		{"expiry beyond date", ClientMessage{Type: "update_partner_invite", ExpiresAt: maxInviteExpiry + 1}, false},
		{"expiry past", ClientMessage{Type: "update_partner_invite", ExpiresAt: time.Now().Unix() - 1}, false},
		{"duration exact", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: 300}}, true},
		{"duration above limit", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: 301}}, false},
		{"duration unknown", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: 0}}, true},
		{"duration negative", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: -1}}, false},
		{"track metadata", ClientMessage{Type: "enqueue", Track: &TrackData{Title: strings.Repeat("x", 513)}}, false},
		{"track link", ClientMessage{Type: "enqueue", Track: &TrackData{URL: strings.Repeat("x", 4097)}}, false},
		{"recommendation count", ClientMessage{Type: "mix", Count: 25}, true},
		{"excessive recommendations", ClientMessage{Type: "mix", Count: 26}, false},
		{"negative recommendations", ClientMessage{Type: "mix", Count: -1}, false},
		{"repeat exact", ClientMessage{Type: "loop", Count: 10}, true},
		{"repeat excessive", ClientMessage{Type: "loop", Count: 11}, false},
		{"seek infinite", ClientMessage{Type: "seek", PositionSeconds: math.Inf(1)}, false},
		{"seek nan", ClientMessage{Type: "seek", PositionSeconds: math.NaN()}, false},
		{"seek negative", ClientMessage{Type: "seek", PositionSeconds: -1}, false},
		{"queue invalid index", ClientMessage{Type: "remove", Index: 0}, false},
		{"queue excessive index", ClientMessage{Type: "remove", Index: 11}, false},
		{"oversized ID", ClientMessage{Type: "get_playlists", PlaylistID: strings.Repeat("x", 257)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateClientMessage(tc.msg); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
	for _, duration := range []int32{-1, 0, 299, 300, 301, math.MaxInt32} {
		if allowedTrackDuration(duration) != (duration >= 0 && duration <= 300) {
			t.Fatalf("unexpected duration filter for %d", duration)
		}
	}
}

func TestInvalidNumericMessagesKeepRequestIdentity(t *testing.T) {
	for _, value := range []string{"5555555555555556649495956596595629592929595929", "1e999", "1.5", "null"} {
		raw := `{"type":"update_partner_invite","requestId":"limit-42","userLimit":` + value + `}`
		if _, err := decodeClientMessage(raw); err == nil {
			t.Fatalf("numeric overflow or fraction decoded: %s", value)
		}
		client := voiceTestClient(42, 42, true)
		sendInvalidClientMessage(client, raw)
		var event struct{ Event, Command, RequestID, Data string }
		if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
			t.Fatal(err)
		}
		if event.Event != "error" || event.Command != "update_partner_invite" || event.RequestID != "limit-42" || event.Data == "" {
			t.Fatalf("uncorrelated invalid request: %+v", event)
		}
	}
}

func TestInvalidInviteUpdatesPreserveCurrentSettings(t *testing.T) {
	store := &partnerInviteStore{links: make(map[string]*partnerInvite)}
	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	token, err := store.create(42, expiry, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{-1, 1001, math.MaxInt} {
		if _, err := store.create(42, expiry, limit); err == nil {
			t.Fatalf("accepted user limit %d", limit)
		}
		_, savedExpiry, savedLimit := store.current(42)
		if savedExpiry != expiry.Unix() || savedLimit != 3 {
			t.Fatal("invalid update mutated the invite")
		}
	}
	if _, err := store.create(42, time.Unix(maxInviteExpiry+1, 0), 1); err == nil {
		t.Fatal("accepted unsupported expiry")
	}
	if store.links[token] == nil {
		t.Fatal("invalid settings invalidated the current link")
	}
}

func TestSearchRejectsOversizedUnicodeQuery(t *testing.T) {
	previous := config.Token
	config.Token = "input-limit-test-token"
	t.Cleanup(func() { config.Token = previous })
	r := httptest.NewRequest(http.MethodGet, "/api/search?q="+url.QueryEscape(strings.Repeat("界", 201)), nil)
	r.Header.Set("X-Telegram-Init-Data", partnerInitData(42, "42"))
	response := httptest.NewRecorder()
	searchHandler(response, r)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("query status=%d, body=%s", response.Code, response.Body.String())
	}
}
