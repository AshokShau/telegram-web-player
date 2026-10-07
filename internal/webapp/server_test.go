package webapp

import (
	"ashokshau/tg-web/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicPages(t *testing.T) {
	previous := config.SupportGroup
	config.SupportGroup = "https://t.me/test_support?name=a&other=b"
	t.Cleanup(func() { config.SupportGroup = previous })

	for _, tc := range []struct {
		path    string
		handler http.HandlerFunc
		title   string
	}{
		{"/", ServeHomeHTML, "SyncTune · Listen together"},
		{"/privacy", ServePrivacyHTML, "Privacy policy · SyncTune"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			tc.handler(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}
			if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatal("page was not served as HTML")
			}
			body := response.Body.String()
			if !strings.Contains(body, tc.title) || !strings.Contains(body, "https://t.me/test_support?name=a&amp;other=b") || strings.Contains(body, "{{") {
				t.Fatal("page content or configured support link did not render")
			}
			missing := httptest.NewRecorder()
			tc.handler(missing, httptest.NewRequest(http.MethodGet, tc.path+"/missing", nil))
			if missing.Code != http.StatusNotFound {
				t.Fatalf("unknown path status = %d", missing.Code)
			}
		})
	}
}
