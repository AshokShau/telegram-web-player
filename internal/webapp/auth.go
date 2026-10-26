/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/telegram-web-player
 */

package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"errors"
	"slices"
	"time"

	td "github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/logger"
	tgwebapp "github.com/AshokShau/gotdbot/webapp"
)

var log = logger.New()

func verifyTelegramInitData(initDataRaw string, botToken string) (*tgwebapp.WebAppInitData, bool) {
	data, err := validateTelegramInitData(initDataRaw, botToken)
	return data, err == nil
}

func validateTelegramInitData(initDataRaw string, botToken string) (*tgwebapp.WebAppInitData, error) {
	if initDataRaw == "" {
		return nil, errors.New("missing_init_data")
	}

	const clockSkew = 5 * time.Minute
	data, err := tgwebapp.ValidateTelegramInitData(initDataRaw, botToken, &tgwebapp.ValidationOptions{
		// The library adds ClockSkew to MaxAge; keep a strict 24-hour age limit.
		MaxAge: 24*time.Hour - clockSkew, ClockSkew: clockSkew,
		Now: time.Now().Truncate(time.Second),
	})
	if err != nil {
		reason := "malformed_init_data"
		switch {
		case errors.Is(err, tgwebapp.ErrBotTokenRequired):
			reason = "missing_bot_token"
		case errors.Is(err, tgwebapp.ErrInvalidHash):
			reason = "hash_mismatch"
		case errors.Is(err, tgwebapp.ErrAuthDate):
			reason = "invalid_auth_date"
		case errors.Is(err, tgwebapp.ErrExpired):
			reason = "expired_auth_date"
		case errors.Is(err, tgwebapp.ErrFutureAuthDate):
			reason = "future_auth_date"
		}
		return nil, errors.New(reason)
	}

	if data.User == nil || data.User.ID <= 0 {
		return nil, errors.New("invalid_user")
	}

	return data, nil
}

func isUserRoomAdmin(bot *td.Client, chatID int64, userID int64) bool {
	if userID == 0 {
		return false
	}
	if slices.Contains(config.DEVS, userID) {
		return true
	}

	if chatID > 0 {
		return chatID == userID
	}
	if db.Instance != nil && db.Instance.IsAdmin(chatID, userID) {
		return true
	}

	if bot == nil {
		log.Warnf("bot not found for user %d", userID)
		return false
	}

	admins, err := cache.GetAdmins(bot, chatID, false)
	if err != nil || len(admins) == 0 {
		return false
	}

	for _, admin := range admins {
		var memberID int64
		switch m := admin.MemberId.(type) {
		case td.MessageSenderUser:
			memberID = m.UserId
		case *td.MessageSenderUser:
			memberID = m.UserId
		case td.MessageSenderChat:
			memberID = m.ChatId
		case *td.MessageSenderChat:
			memberID = m.ChatId
		}

		if memberID == userID {
			return true
		}
	}

	return false
}

// Administrative settings authority is independent from playback mode.
type roomPermissions struct {
	IsAdmin             bool
	IsAuth              bool
	CanControl          bool
	CanPlay             bool
	CanChangeSilentMode bool
}

func permissionsForRole(admin, auth bool, adminMode string, restrictedPlay bool) roomPermissions {
	return roomPermissions{IsAdmin: admin, IsAuth: auth,
		CanControl: admin || auth || adminMode == utils.Everyone,
		CanPlay:    admin || auth || !restrictedPlay}
}

func resolveRoomPermissions(bot *td.Client, chatID, userID int64) roomPermissions {
	if userID == 0 || (chatID > 0 && chatID != userID) {
		return roomPermissions{}
	}
	admin := isUserRoomAdmin(bot, chatID, userID)
	permissions := permissionsForRole(admin, false, utils.Admins, true)
	if chatID < 0 && db.Instance != nil {
		permissions = permissionsForRole(admin, !admin && db.Instance.IsAuthUser(chatID, userID), db.Instance.GetAdminMode(chatID), db.Instance.GetPlayMode(chatID))
	}
	if chatID < 0 && bot != nil {
		permissions.CanChangeSilentMode = cache.SettingsAccess(bot, chatID, userID, "silent") == nil
	}
	return permissions
}
