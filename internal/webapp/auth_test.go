package webapp

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"encoding/json"
	"strings"
	"testing"
)

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
