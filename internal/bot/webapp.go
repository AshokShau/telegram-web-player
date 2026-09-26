/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

func webAppCommandHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	botUsername := c.Me.Usernames.EditableUsername

	text := "🌐 <b>Web App Player</b>\n\nClick the button below to open the synchronized Web App player for this chat room."
	_, err := m.ReplyText(c, text, &td.SendTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: utils.WebAppControlButtons("webapp", botUsername, chatID),
	})

	return err
}
