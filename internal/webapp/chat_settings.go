package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"errors"
	"log/slog"

	td "github.com/AshokShau/gotdbot"
)

type chatSettingsStore interface {
	SetChatEnabled(int64, bool) error
	SetChatCooldown(int64, int) error
	SetPlayMode(int64, bool) error
	SetAdminMode(int64, string) error
	SetCmdDelete(int64, bool) error
	SetAutoDelete(int64, bool) error
	SetAutoDeleteMinutes(int64, int) error
	SetSilentMode(int64, bool) error
}

func updateChatSettings(bot *td.Client, chatID, userID int64, msg ClientMessage, store chatSettingsStore, cancelCleanup func(int64)) error {
	supplied := 0
	for _, present := range []bool{msg.ChatEnabled != nil, msg.ChatCooldown != nil, msg.RestrictedPlay != nil, msg.AdminMode != nil, msg.CmdDelete != nil, msg.AutoDelete != nil, msg.AutoDeleteMinutes != nil, msg.SilentMode != nil, msg.Autoplay != nil} {
		if present {
			supplied++
		}
	}
	if supplied == 0 {
		return errors.New("Choose a setting to update.")
	}
	if supplied != 1 {
		return errors.New("Change one setting at a time.")
	}
	if msg.AutoDeleteMinutes != nil {
		if err := db.ValidateAutoDeleteMinutes(*msg.AutoDeleteMinutes); err != nil {
			return err
		}
	}
	if msg.AdminMode != nil && *msg.AdminMode != utils.Admins && *msg.AdminMode != utils.Everyone {
		return errors.New("Choose Everyone or Admins for command access.")
	}
	if msg.ChatCooldown != nil && (*msg.ChatCooldown < 0 || *msg.ChatCooldown > 60) {
		return errors.New("Slow mode must be between 0 and 60 seconds.")
	}
	if chatID >= 0 {
		if msg.ChatEnabled == nil && msg.ChatCooldown == nil {
			return errors.New("These settings are available only in groups.")
		}
		if chatID != userID {
			return errors.New("You cannot change another user's room settings.")
		}
	} else {
		action := "chat"
		if msg.SilentMode != nil {
			action = "silent"
		}
		if err := cache.SettingsAccess(bot, chatID, userID, action); err != nil {
			return err
		}
	}
	var err error
	switch {
	case msg.ChatEnabled != nil:
		err = store.SetChatEnabled(chatID, *msg.ChatEnabled)
	case msg.ChatCooldown != nil:
		err = store.SetChatCooldown(chatID, *msg.ChatCooldown)
	case msg.RestrictedPlay != nil:
		err = store.SetPlayMode(chatID, *msg.RestrictedPlay)
	case msg.AdminMode != nil:
		err = store.SetAdminMode(chatID, *msg.AdminMode)
	case msg.CmdDelete != nil:
		if *msg.CmdDelete && !cache.BotCanDeleteCommands(bot, chatID) {
			return errors.New("The bot needs Delete Messages permission to delete users' commands.")
		}
		err = store.SetCmdDelete(chatID, *msg.CmdDelete)
	case msg.AutoDelete != nil:
		err = store.SetAutoDelete(chatID, *msg.AutoDelete)
	case msg.AutoDeleteMinutes != nil:
		err = store.SetAutoDeleteMinutes(chatID, *msg.AutoDeleteMinutes)
	case msg.SilentMode != nil:
		err = store.SetSilentMode(chatID, *msg.SilentMode)
	case msg.Autoplay != nil:
		if !cache.ChatCache.IsActive(chatID) {
			return errors.New("Bot is not streaming.")
		}
		cache.ChatCache.SetAutoplay(chatID, *msg.Autoplay)
	}
	if err != nil {
		slog.Warn("web-app settings update failed", "chat_id", chatID)
		return errors.New("Unable to save settings. Please try again.")
	}
	if msg.AutoDelete != nil && !*msg.AutoDelete && cancelCleanup != nil {
		cancelCleanup(chatID)
	}
	return nil
}
