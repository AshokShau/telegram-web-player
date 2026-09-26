/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/cache"
	"fmt"
	"time"

	"github.com/AshokShau/gotdbot"
)

const reloadCooldown = 2 * time.Minute

var reloadRateLimit = cache.NewCache[time.Time](reloadCooldown)

func reloadAdminCacheHandler(c *gotdbot.Client, m *gotdbot.Message) error {
	if m.IsPrivate() {
		return gotdbot.EndGroups
	}

	now := time.Now()
	reloadKey := fmt.Sprintf("reload:%d", m.ChatId)

	if lastUsed, ok := reloadRateLimit.Get(reloadKey); ok {
		if elapsed := now.Sub(lastUsed); elapsed < reloadCooldown {
			remaining := int32((reloadCooldown - elapsed).Seconds())
			_, _ = m.ReplyText(c, fmt.Sprintf("Please wait %s before using this command again.", secondsToMinutes(remaining)), nil)
			return nil
		}
	}

	reloadRateLimit.Set(reloadKey, now)

	reply, err := m.ReplyText(c, "Reloading administrator cache...", nil)
	if err != nil {
		c.Logger.Warn(
			"Failed to send reloading message",
			"chat_id", m.ChatId,
			"error", err,
		)
		return gotdbot.EndGroups
	}

	cache.ClearAdminCache(m.ChatId)

	admins, err := cache.GetAdmins(c, m.ChatId, true)
	if err != nil {
		c.Logger.Warn(
			"Failed to reload administrator cache",
			"chat_id", m.ChatId,
			"error", err,
		)
		_, _ = reply.EditText(c, "Failed to reload administrator cache.", nil)
		return gotdbot.EndGroups
	}

	c.Logger.Info("Reloaded administrator cache", "chat_id", m.ChatId, "count", len(admins))
	_, _ = reply.EditText(c, "Administrator cache reloaded successfully.", nil)
	return gotdbot.EndGroups
}

func secondsToMinutes(secs int32) string {
	m := secs / 60
	s := secs % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// privacyHandler handles the /privacy command.
func privacyHandler(c *gotdbot.Client, m *gotdbot.Message) error {
	botName := c.Me.FirstName

	text := fmt.Sprintf(
		`<b>Privacy Policy for %s</b>

<b>1. Data Storage:</b>
We do not store personal data on your device or track your browsing activity.

<b>2. Collection:</b>
We only collect your Telegram <b>User ID</b> and <b>Chat ID</b> when required to provide bot functionality. We do not intentionally store names, phone numbers, or locations.

<b>3. Usage:</b>
Collected data is used strictly for bot functionality and is not used for marketing or commercial purposes.

<b>4. Sharing:</b>
We do not sell, trade, or share collected data with third parties.

<b>5. Security:</b>
We use reasonable security measures to protect stored data. However, no online service can guarantee 100%% security.

<b>6. Cookies:</b>
%s does not use cookies or web tracking technologies.

<b>7. Third Parties:</b>
We do not integrate with third-party data collectors, other than Telegram itself and services required for the bot to function.

<b>8. Your Rights:</b>
You can request deletion of your stored data or block the bot to stop further interaction.

<b>9. Updates:</b>
Changes to this privacy policy will be announced through the bot.

<b>10. Contact:</b>
Questions? Contact our <a href="https://t.me/GuardxSupport">Support Group</a>.

──────────────────
<b>Note:</b> This policy is intended to provide a safe and respectful experience with %s.`,
		botName,
		botName,
		botName,
	)

	_, err := m.ReplyText(c, text, &gotdbot.SendTextMessageOpts{
		ParseMode:             "html",
		DisableWebPagePreview: true,
	})

	return err
}
