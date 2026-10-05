package bot

import (
	"ashokshau/tg-web/internal/config"
	"fmt"
	"html"

	"github.com/AshokShau/gotdbot"
)

func privacyText(botName, policyURL, supportURL string) string {
	policyLink := "Read the full policy in the player: <b>Profile → Privacy policy</b>."
	if policyURL != "" {
		policyLink = fmt.Sprintf(`Read the <a href="%s">full privacy policy</a>.`, html.EscapeString(policyURL))
	}
	return fmt.Sprintf(`<b>Privacy · %s</b>

<b>Data we process</b>
Telegram IDs, available profile details and authentication data; commands, searches, submitted media, playlists and room settings. Connection and diagnostic logs may contain IDs, IP addresses and error details.

<b>Shared rooms</b>
Your displayed profile, presence, track requests and room messages are visible to other listeners. Recent room chat stays in server memory (up to 50 messages). Voice audio is relayed through our server to participants; SyncTune does not record it. Participants may make their own copies or recordings.

<b>Storage and services</b>
IDs, playlists, permissions and settings are stored in the database until removed. The player saves preferences and recent tracks on your device. Telegram, hosting, media services, CDNs and Google's STUN service process data needed to deliver their features.

<b>Your choices</b>
Microphone access requires permission. Bot message access is requested separately. Delete playlists in the player, clear site storage to remove local data, or ask the operator for access, correction or deletion. Blocking the bot stops contact but does not erase stored data.

%s
<a href="%s">Contact support</a> and ask for a private conversation with the operator for data requests. Do not post sensitive details in a public group.`, html.EscapeString(botName), policyLink, html.EscapeString(supportURL))
}

func privacyHandler(c *gotdbot.Client, m *gotdbot.Message) error {
	_, err := m.ReplyText(c, privacyText(c.Me.FirstName, config.PrivacyPolicyURL, config.SupportGroup), &gotdbot.SendTextMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	return err
}
