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
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/messaging"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"html"
	"strconv"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func settingsHandler(c *td.Client, m *td.Message) error {
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChatId
	chatInfo, chatErr := m.GetChat(c)
	if chatErr != nil || !messaging.IsGroup(chatInfo) {
		return nil
	}

	deleteCmd(c, m)

	if accessErr := cache.SettingsAccess(c, chatID, m.SenderID(), "main"); accessErr != nil {
		_, _ = m.ReplyText(c, accessErr.Error(), nil)
		return td.EndGroups
	}
	if _, err := db.Instance.GetCleanSettings(chatID); err != nil {
		_, _ = m.ReplyText(c, "Unable to load settings. Please try again later.", nil)
		return td.EndGroups
	}
	var err error

	// Get current settings
	getPlayMode := db.Instance.GetPlayMode(chatID)
	playModeStr := utils.Everyone
	if getPlayMode {
		playModeStr = utils.Admins
	}
	getAdminMode := db.Instance.GetAdminMode(chatID)
	cmdDelete := db.Instance.GetCmdDelete(chatID)
	language, _ := db.Instance.GetLanguage(chatID)
	autoplay := cache.ChatCache.GetAutoplay(chatID)
	chatEnabled := db.Instance.GetChatEnabled(chatID)

	chat, err := m.GetChat(c)
	if err != nil {
		c.Logger.Warn("Failed to get chat", "error", err)
		return nil
	}

	text := fmt.Sprintf("<u><b>%s settings</b></u>\n\nClick the buttons below to change this chat's current settings.",
		html.EscapeString(chat.Title))

	_, err = m.ReplyText(c, text, &td.SendMessageOpts{ReplyMarkup: cleanSettingsKeyboard(chatID, playModeStr, getAdminMode, cmdDelete, language, autoplay, chatEnabled), ParseMode: td.ParseModeHTML})
	return err
}

func settingsCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery, cancelCleanup func(int64)) error {
	if cb.IsPrivate() {
		return nil
	}

	chatID := cb.ChatId

	chat, err := c.GetChat(chatID)
	if err != nil || !messaging.IsGroup(chat) {
		return cb.Answer(c, 0, true, "These settings are available only in groups.", "")
	}
	if accessErr := cache.SettingsAccess(c, chatID, cb.SenderUserId, "main"); accessErr != nil {
		return cb.Answer(c, 0, true, accessErr.Error(), "")
	}

	data := cb.DataString()
	if data == "settings_main" {
		return cb.Answer(c, 0, false, "Update your chat settings", "")
	}

	parts := strings.Split(data, "_")
	if len(parts) < 2 || parts[0] != "settings" {
		return cb.Answer(c, 0, true, "Invalid settings button. Open /settings again.", "")
	}
	settingType := parts[1]
	if accessErr := cache.SettingsAccess(c, chatID, cb.SenderUserId, settingType); accessErr != nil {
		return cb.Answer(c, 0, true, accessErr.Error(), "")
	}
	_, err = db.Instance.GetCleanSettings(chatID)
	if err != nil {
		return cb.Answer(c, 0, true, "Unable to load settings. Try again later.", "")
	}
	if settingType == "autodelete" || settingType == "silent" || settingType == "timer" {
		value, parseErr := parseCleanCallback(data)
		if parseErr != nil {
			return cb.Answer(c, 0, true, parseErr.Error(), "")
		}
		switch settingType {
		case "autodelete":
			err = db.Instance.SetAutoDelete(chatID, value == 1)
			if err == nil && value == 0 && cancelCleanup != nil {
				cancelCleanup(chatID)
			}
		case "silent":
			err = db.Instance.SetSilentMode(chatID, value == 1)
		case "timer":
			err = db.Instance.SetAutoDeleteMinutes(chatID, value)
		}
	} else if len(parts) != 2 {
		return cb.Answer(c, 0, true, "This settings button is outdated. Open /settings again.", "")
	} else {
		switch settingType {
		case "delete":
			cmdDelete := db.Instance.GetCmdDelete(chatID)
			if !cmdDelete {
				if !cache.BotCanDeleteCommands(c, chatID) {
					_ = cb.Answer(c, 0, true, "I don't have permission to delete messages in this chat.", "")
					return nil
				}
				err = db.Instance.SetCmdDelete(chatID, true)
			} else {
				err = db.Instance.SetCmdDelete(chatID, false)
			}
		case "play":
			getPlayMode := db.Instance.GetPlayMode(chatID)
			err = db.Instance.SetPlayMode(chatID, !getPlayMode)
		case "admin":
			getAdminMode := db.Instance.GetAdminMode(chatID)
			newMode := utils.Everyone
			if getAdminMode == utils.Everyone {
				newMode = utils.Admins
			}
			err = db.Instance.SetAdminMode(chatID, newMode)
		case "autoplay":
			if !cache.ChatCache.IsActive(chatID) {
				_ = cb.Answer(c, 0, true, "Bot is not streaming.", "")
				return nil
			}
			autoplay := cache.ChatCache.GetAutoplay(chatID)
			cache.ChatCache.SetAutoplay(chatID, !autoplay)
		case "chat":
			chatEnabled := db.Instance.GetChatEnabled(chatID)
			newEnabled := !chatEnabled
			err = db.Instance.SetChatEnabled(chatID, newEnabled)
		case "lang":
			return cb.Answer(c, 0, true, "Language selection is not yet implemented via this menu.", "")
		default:
			return cb.Answer(c, 0, true, "Unknown setting", "")
		}

	}
	if err != nil {
		return cb.Answer(c, 0, true, "Unable to save settings. Please try again.", "")
	}
	webapp.HubInstance.BroadcastChatSettings(c, chatID)

	getPlayMode := db.Instance.GetPlayMode(chatID)
	playModeStr := utils.Everyone
	if getPlayMode {
		playModeStr = utils.Admins
	}
	getAdminMode := db.Instance.GetAdminMode(chatID)
	cmdDelete := db.Instance.GetCmdDelete(chatID)
	language, _ := db.Instance.GetLanguage(chatID)
	autoplay := cache.ChatCache.GetAutoplay(chatID)
	chatEnabled := db.Instance.GetChatEnabled(chatID)

	chat, err = c.GetChat(chatID)
	if err != nil {
		c.Logger.Warn("Failed to get chat", "error", err)
		return nil
	}

	text := fmt.Sprintf("<u><b>%s settings</b></u>\n\nClick the buttons below to change this chat's current settings.",
		html.EscapeString(chat.Title))

	_, err = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: cleanSettingsKeyboard(chatID, playModeStr, getAdminMode, cmdDelete, language, autoplay, chatEnabled), ParseMode: td.ParseModeHTML})
	if err != nil {
		return err
	}

	_ = cb.Answer(c, 0, false, "Settings updated", "")
	return nil
}

func parseCleanCallback(data string) (int, error) {
	parts := strings.Split(data, "_")
	if len(parts) != 3 || parts[0] != "settings" {
		return 0, fmt.Errorf("invalid settings button. Open /settings again")
	}
	value, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid setting value")
	}
	switch parts[1] {
	case "timer":
		if err := db.ValidateAutoDeleteMinutes(value); err != nil {
			return 0, err
		}
	case "autodelete", "silent":
		if value != 0 && value != 1 {
			return 0, fmt.Errorf("invalid setting value")
		}
	default:
		return 0, fmt.Errorf("unknown setting")
	}
	return value, nil
}
func cleanSettingsKeyboard(chatID int64, playMode, adminMode string, cmdDelete bool, language string, autoplay, chatEnabled bool) *td.ReplyMarkupInlineKeyboard {
	keyboard := utils.SettingsKeyboard(playMode, adminMode, cmdDelete, language, autoplay, chatEnabled)
	settings, err := db.Instance.GetCleanSettings(chatID)
	if err != nil {
		return keyboard
	}
	button := func(text, data string) td.InlineKeyboardButton {
		return td.InlineKeyboardButton{Text: text, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(data)}}
	}
	state := func(on bool) string {
		if on {
			return "ON"
		}
		return "OFF"
	}
	next := func(on bool) int {
		if on {
			return 0
		}
		return 1
	}
	rows := [][]td.InlineKeyboardButton{
		{button("Auto-delete: "+state(settings.AutoDelete), fmt.Sprintf("settings_autodelete_%d", next(settings.AutoDelete)))},
		{button(fmt.Sprintf("Delete after: %d minutes", settings.Minutes), "settings_main")},
	}
	var timers []td.InlineKeyboardButton
	for minutes := 2; minutes <= 7; minutes++ {
		label := strconv.Itoa(minutes)
		if minutes == settings.Minutes {
			label = "✓ " + label
		}
		timers = append(timers, button(label, fmt.Sprintf("settings_timer_%d", minutes)))
	}

	rows = append(rows, timers, []td.InlineKeyboardButton{button("Silent mode: "+state(settings.Silent), fmt.Sprintf("settings_silent_%d", next(settings.Silent)))})
	keyboard.Rows = append(rows, keyboard.Rows...)
	return keyboard
}
