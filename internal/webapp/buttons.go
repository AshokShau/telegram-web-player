package webapp

import (
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

const playerContextCallbackPrefix = "webapp_verify:"

func WebAppControlButtons(mode, botUsername string, chatID int64) *td.ReplyMarkupInlineKeyboard {
	instance := ""
	if chatID < 0 {
		instance, _ = db.Instance.GetTelegramChatInstance(chatID)
	}
	return webAppControlButtonsForContext(mode, botUsername, chatID, instance)
}

func webAppControlButtonsForContext(mode, botUsername string, chatID int64, instance string) *td.ReplyMarkupInlineKeyboard {
	markup := utils.WebAppControlButtons(mode, botUsername, chatID)
	if chatID >= 0 || instance != "" {
		return markup
	}

	markup.Rows[len(markup.Rows)-1][0].Type = &td.InlineKeyboardButtonTypeCallback{
		Data: []byte(playerContextCallbackPrefix + mode),
	}
	markup.Rows[len(markup.Rows)-1][0].Text = "🔐 Verify group for player"
	return markup
}

func PlayerContextCallbackMode(data string) (string, bool) {
	if data == "webapp_verify" {
		return "webapp", true
	}

	if len(data) < len(playerContextCallbackPrefix) || data[:len(playerContextCallbackPrefix)] != playerContextCallbackPrefix {
		return "", false
	}

	mode := data[len(playerContextCallbackPrefix):]
	switch mode {
	case "", "play", "pause", "resume", "webapp":
		return mode, true
	default:
		log.Warnf("playerContextCallbackMode: invalid mode: %s", mode)
		return "", false
	}
}
