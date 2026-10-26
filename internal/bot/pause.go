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
	"ashokshau/tg-web/internal/webapp"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func pauseHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "There is no active playback.", nil)
		return nil
	}

	if _, err := webapp.Manager.Pause(c, chatID); err != nil {
		_, _ = m.ReplyText(c, "Failed to pause the playback: Please try again later.", nil)
		return nil
	}

	markup := webapp.WebAppControlButtons("pause", c.Me().Usernames.EditableUsername, chatID)
	_, err := messaging.Reply(c, m, fmt.Sprintf("Playback has been paused by %s.", firstName(c, m)), &td.SendMessageOpts{ReplyMarkup: markup})
	return err
}

func resumeHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "There is no active playback.", nil)
		return nil
	}

	if _, err := webapp.Manager.Resume(c, chatID); err != nil {
		_, _ = m.ReplyText(c, "Failed to resume the playback: Please try again later.", nil)
		return nil
	}

	markup := webapp.WebAppControlButtons("resume", c.Me().Usernames.EditableUsername, chatID)
	_, err := messaging.Reply(c, m, fmt.Sprintf("Playback has been resumed by %s.", firstName(c, m)), &td.SendMessageOpts{ReplyMarkup: markup})
	return err
}
