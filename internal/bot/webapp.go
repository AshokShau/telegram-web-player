/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"strconv"

	td "github.com/AshokShau/gotdbot"
)

func webAppCommandHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	botUsername := c.Me.Usernames.EditableUsername

	text := "🌐 <b>Web App Player</b>\n\nClick the button below to open the synchronized Web App player for this chat room."
	_, err := m.ReplyText(c, text, &td.SendTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: webapp.WebAppControlButtons("webapp", botUsername, chatID),
	})

	return err
}

func webAppVerifyCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	mode, valid := webapp.PlayerContextCallbackMode(cb.DataString())
	if !valid {
		return cb.Answer(c, 0, true, "Open a fresh player message from this group.", "")
	}

	if cb.ChatId >= 0 || cb.ChatInstance == 0 {
		return cb.Answer(c, 0, true, "Open the player from the Telegram group you want to use.", "")
	}

	instance, err := db.Instance.GetTelegramChatInstance(cb.ChatId)
	if err != nil || instance != strconv.FormatInt(cb.ChatInstance, 10) {
		return cb.Answer(c, 0, true, "Unable to verify this group right now. Try again later.", "")
	}

	_, err = c.EditMessageReplyMarkup(cb.ChatId, cb.MessageId, &td.EditMessageReplyMarkupOpts{
		ReplyMarkup: utils.WebAppControlButtons(mode, c.Me.Usernames.EditableUsername, cb.ChatId),
	})

	if err != nil {
		c.Logger.Warnf("webAppVerifyCallbackHandler: Error updating message: %v", err)
		_ = cb.Answer(c, 0, true, "Group verified. Open a fresh bot player message.", "")
		return nil
	}

	return cb.Answer(c, 0, false, "Group ready. Tap Open Web App Player to join.", "")
}
