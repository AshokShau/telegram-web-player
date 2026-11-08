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
)

func TestClientMessageInputBoundaries(t *testing.T) {
	previous := config.SongDurationLimit
	config.SongDurationLimit = 300
	t.Cleanup(func() { config.SongDurationLimit = previous })
	collectionTracks := make([]*TrackData, 101)
	for i := range collectionTracks[:100] {
		collectionTracks[i] = &TrackData{ID: "abcdefghijk", URL: "https://www.youtube.com/watch?v=abcdefghijk", Title: "Song", Platform: "youtube", Duration: 180}
	}
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
		{"duration exact", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: 300}}, true},
		{"duration above limit", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: 301}}, false},
		{"duration unknown", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: 0}}, true},
		{"duration negative", ClientMessage{Type: "enqueue", Track: &TrackData{Duration: -1}}, false},
		{"collection play", ClientMessage{Type: "play_collection", Tracks: []*TrackData{{ID: "abcdefghijk", URL: "https://www.youtube.com/watch?v=abcdefghijk", Title: "Song", Platform: "youtube", Duration: 180}}}, true},
		{"collection empty", ClientMessage{Type: "play_collection"}, false},
		{"collection extra ignored", ClientMessage{Type: "play_collection", Tracks: collectionTracks}, true},
		{"collection bad entry skipped", ClientMessage{Type: "play_collection", Tracks: []*TrackData{{ID: "bad"}, {ID: "abcdefghijk", URL: "https://www.youtube.com/watch?v=abcdefghijk", Title: "Song", Platform: "youtube", Duration: 180}}}, true},
		{"collection missing URL", ClientMessage{Type: "play_collection", Tracks: []*TrackData{{ID: "abcdefghijk", Title: "Song", Platform: "youtube", Duration: 180}}}, false},
		{"collection mismatched URL", ClientMessage{Type: "play_collection", Tracks: []*TrackData{{ID: "abcdefghijk", URL: "https://www.youtube.com/watch?v=wrongvideo", Title: "Song", Platform: "youtube", Duration: 180}}}, false},
		{"collection invalid video", ClientMessage{Type: "play_collection", Tracks: []*TrackData{{ID: "bad", Title: "Song", Platform: "youtube", Duration: 180}}}, false},
		{"collection wrong platform", ClientMessage{Type: "play_collection", Tracks: []*TrackData{{ID: "abcdefghijk", URL: "https://www.youtube.com/watch?v=abcdefghijk", Title: "Song", Platform: "spotify", Duration: 180}}}, false},
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
		{"queue index beyond old limit", ClientMessage{Type: "remove", Index: 101}, true},
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
		raw := `{"type":"loop","requestId":"limit-42","count":` + value + `}`
		if _, err := decodeClientMessage(raw); err == nil {
			t.Fatalf("numeric overflow or fraction decoded: %s", value)
		}
		client := voiceTestClient(42, 42, true)
		sendInvalidClientMessage(client, raw)
		var event struct{ Event, Command, RequestID, Data string }
		if err := json.Unmarshal([]byte(<-client.outbox), &event); err != nil {
			t.Fatal(err)
		}
		if event.Event != "error" || event.Command != "loop" || event.RequestID != "limit-42" || event.Data == "" {
			t.Fatalf("uncorrelated invalid request: %+v", event)
		}
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
