package bot

import (
	"ashokshau/tg-web/internal/db"
	"errors"
	"strconv"

	td "github.com/AshokShau/gotdbot"
)

type telegramChatInstanceBinder interface {
	BindTelegramChatInstance(chatID int64, instance string) error
}

func rememberTelegramChatContext(cb *td.UpdateNewCallbackQuery, store telegramChatInstanceBinder) error {
	if cb == nil || cb.ChatId >= 0 || cb.ChatInstance == 0 {
		return nil
	}
	if store == nil {
		return errors.New("chat_binding_unavailable")
	}

	return store.BindTelegramChatInstance(cb.ChatId, strconv.FormatInt(cb.ChatInstance, 10))
}

func recordTelegramChatContext(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if err := rememberTelegramChatContext(cb, db.Instance); err != nil {
		c.Logger.Warn("Unable to save Telegram chat context", "chat_id", cb.ChatId, "reason", "chat_binding_failed")
	}

	return td.ContinueGroups
}
