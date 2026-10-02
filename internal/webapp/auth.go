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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/logger"
)

var log = logger.New()

type WebAppUser struct {
	ID                    int64  `json:"id"`
	IsBot                 bool   `json:"is_bot,omitempty"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name,omitempty"`
	Username              string `json:"username,omitempty"`
	LanguageCode          string `json:"language_code,omitempty"`
	IsPremium             bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu bool   `json:"added_to_attachment_menu,omitempty"`
	AllowsWriteToPM       bool   `json:"allows_write_to_pm,omitempty"`
	PhotoURL              string `json:"photo_url,omitempty"`
}

type WebAppInitData struct {
	User     *WebAppUser `json:"user,omitempty"`
	AuthDate int64       `json:"auth_date"`
	Hash     string      `json:"hash"`
}

func verifyTelegramInitData(initDataRaw string, botToken string) (*WebAppInitData, bool) {
	if initDataRaw == "" {
		return nil, false
	}

	values, err := url.ParseQuery(initDataRaw)
	if err != nil {
		return nil, false
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, false
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	dataCheckArr := make([]string, 0, len(keys))
	for _, k := range keys {
		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(dataCheckArr, "\n")

	macSecret := hmac.New(sha256.New, []byte("WebAppData"))
	macSecret.Write([]byte(botToken))
	secretKey := macSecret.Sum(nil)

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(calculatedHash), []byte(hash)) {
		return nil, false
	}

	initData := &WebAppInitData{
		Hash: hash,
	}

	if ad := values.Get("auth_date"); ad != "" {
		fmt.Sscanf(ad, "%d", &initData.AuthDate)
	}

	if initData.AuthDate == 0 {
		return nil, false
	}

	now := time.Now().Unix()
	const maxAuthAge = 86400 // 24 hours max
	if now-initData.AuthDate > maxAuthAge || initData.AuthDate > now+300 {
		return nil, false
	}

	if userStr := values.Get("user"); userStr != "" {
		var u WebAppUser
		if err = json.Unmarshal([]byte(userStr), &u); err == nil {
			initData.User = &u
		}
	}

	if initData.User == nil || initData.User.ID <= 0 {
		return nil, false
	}

	return initData, true
}

func isUserChatAdmin(bot *td.Client, chatID int64, userID int64) bool {
	if userID == 0 {
		return false
	}

	if slices.Contains(config.DEVS, userID) {
		return true
	}

	if db.Instance.IsAdmin(chatID, userID) {
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

func canUserControl(bot *td.Client, chatID int64, userID int64) bool {
	if userID == 0 {
		return false
	}

	if chatID > 0 {
		return chatID == userID
	}

	if isUserChatAdmin(bot, chatID, userID) {
		return true
	}

	if db.Instance.IsAuthUser(chatID, userID) {
		return true
	}

	adminMode := db.Instance.GetAdminMode(chatID)
	return adminMode == utils.Everyone
}

func canUserPlay(bot *td.Client, chatID int64, userID int64) bool {
	if userID == 0 {
		return false
	}

	if chatID > 0 {
		return chatID == userID
	}

	if isUserChatAdmin(bot, chatID, userID) {
		return true
	}

	if db.Instance.IsAuthUser(chatID, userID) {
		return true
	}

	return !db.Instance.GetPlayMode(chatID)
}
