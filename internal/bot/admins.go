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
	"ashokshau/tg-web/internal/messaging"
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

	reply, err := messaging.Progress(c, m, "Reloading administrator cache...")
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
		if reply.Id != 0 {
			_, _ = reply.EditText(c, "Failed to reload administrator cache.", nil)
		} else {
			_, _ = m.ReplyText(c, "Failed to reload administrator cache.", nil)
		}
		return gotdbot.EndGroups
	}

	c.Logger.Info("Reloaded administrator cache", "chat_id", m.ChatId, "count", len(admins))
	_, _ = messaging.Edit(c, reply, "Administrator cache reloaded successfully.", nil)
	return gotdbot.EndGroups
}

func secondsToMinutes(secs int32) string {
	m := secs / 60
	s := secs % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
