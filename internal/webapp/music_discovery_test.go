package webapp

import (
	"ashokshau/tg-web/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMusicDiscoveryRequiresTelegramAndBoundedSeeds(t *testing.T) {
	old := config.Token
	config.Token = "music-discovery-test-token"
	t.Cleanup(func() { config.Token = old })
	for _, tc := range []struct {
		body string
		auth bool
		want int
	}{
		{`{"seeds":[]}`, false, http.StatusUnauthorized},
		{`{"seeds":[{},{},{},{},{},{},{}]}`, true, http.StatusBadRequest},
		{`{"seeds":[{"title":"` + strings.Repeat("界", 121) + `"}]}`, true, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, "/api/music-discovery", strings.NewReader(tc.body))
		if tc.auth {
			r.Header.Set("X-Telegram-Init-Data", partnerInitData(42, "42"))
		}
		w := httptest.NewRecorder()
		musicDiscoveryHandler(w, r)
		if w.Code != tc.want {
			t.Fatalf("status=%d, want=%d", w.Code, tc.want)
		}
	}
}
