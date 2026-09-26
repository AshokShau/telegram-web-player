/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"fmt"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// Handles the /logger command to toggle logger status
func loggerHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	if config.LoggerId == 0 {
		_, _ = m.ReplyText(c, "Please set LOGGER_ID in .env first.", nil)
		return td.EndGroups
	}

	loggerStatus := db.Instance.GetLoggerStatus()
	args := strings.ToLower(Args(m))
	if len(args) == 0 {
		_, _ = m.ReplyText(c, fmt.Sprintf("Usage: /logger [enable|disable|on|off]\nCurrent status: %t", loggerStatus), nil)
		return td.EndGroups
	}

	switch args {
	case "enable", "on":
		_ = db.Instance.SetLoggerStatus(true)
		_, _ = m.ReplyText(c, "Logger Enabled", nil)
	case "disable", "off":
		_ = db.Instance.SetLoggerStatus(false)
		_, _ = m.ReplyText(c, "Logger disabled", nil)
	default:
		_, _ = m.ReplyText(c, "Invalid argument. Use 'enable', 'disable', 'on', or 'off'.", nil)
	}

	return td.EndGroups
}
