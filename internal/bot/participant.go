/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 */

package bot

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/AshokShau/gotdbot"
)

func handleNewChat(c *gotdbot.Client, update *gotdbot.UpdateNewChat) error {
	chat := update.Chat
	if chat == nil {
		return nil
	}

	switch chat.Type.(type) {
	case *gotdbot.ChatTypeSupergroup:
		if err := db.Instance.AddChat(chat.Id); err != nil {
			c.Logger.Warnf("Failed to add chat to database: chat_id=%d error=%v", chat.Id, err)
		}
	case *gotdbot.ChatTypePrivate:
		if err := db.Instance.AddUser(chat.Id); err != nil {
			c.Logger.Warnf("Failed to add user to database: user_id=%d error=%v", chat.Id, err)
		}
	}

	return nil
}

// handleParticipant processes every chat-member status change for the bot.
func handleParticipant(client *gotdbot.Client, update *gotdbot.UpdateChatMember) error {
	chatID := update.ChatId

	if chatID > 0 {
		return gotdbot.EndGroups
	}

	userID := SenderID(update.NewChatMember.MemberId)

	if userID != client.Me.Id {
		return gotdbot.EndGroups
	}

	chat, err := getSupergroup(client, chatID)
	if err != nil || chat == nil {
		return gotdbot.EndGroups
	}

	go storeChatToDB(chatID)

	oldStatus := update.OldChatMember.Status
	newStatus := update.NewChatMember.Status

	if isAdmin(oldStatus) || isAdmin(newStatus) {
		cache.UpdateAdminCache(chatID, update.NewChatMember)
	}

	if userID == client.Me.Id {
		hasDeleteRights := false
		if s, ok := newStatus.(*gotdbot.ChatMemberStatusAdministrator); ok {
			if s.Rights != nil && s.Rights.CanDeleteMessages {
				hasDeleteRights = true
			}
		}

		if !hasDeleteRights && db.Instance.GetCmdDelete(chatID) {
			_ = db.Instance.SetCmdDelete(chatID, false)
			client.Logger.Info("Bot lost delete message rights; disabled cmd_delete", "chat_id", chatID)
		}
	}

	client.Logger.Debug("Member status changed",
		"user_id", userID,
		"old_status", oldStatus,
		"new_status", newStatus,
		"chat_id", chatID,
	)

	return dispatchStatusChange(client, chatID, userID, oldStatus, newStatus, chat)
}

// dispatchStatusChange routes a member-status transition to the right handler.
func dispatchStatusChange(
	client *gotdbot.Client,
	chatID, userID int64,
	oldStatus, newStatus gotdbot.ChatMemberStatus,
	chat *gotdbot.Supergroup,
) error {
	wasLeft := isStatus[*gotdbot.ChatMemberStatusLeft](oldStatus)
	nowLeft := isStatus[*gotdbot.ChatMemberStatusLeft](newStatus)
	wasMember := isStatus[*gotdbot.ChatMemberStatusMember](oldStatus)
	nowMember := isStatus[*gotdbot.ChatMemberStatusMember](newStatus)
	wasAdmin := isStatus[*gotdbot.ChatMemberStatusAdministrator](oldStatus)
	nowAdmin := isStatus[*gotdbot.ChatMemberStatusAdministrator](newStatus)
	nowBanned := isStatus[*gotdbot.ChatMemberStatusBanned](newStatus)
	wasBanned := isStatus[*gotdbot.ChatMemberStatusBanned](oldStatus)
	wasRestricted := isStatus[*gotdbot.ChatMemberStatusRestricted](oldStatus)
	nowRestricted := isStatus[*gotdbot.ChatMemberStatusRestricted](newStatus)

	switch {
	case wasLeft && (nowMember || nowAdmin || nowRestricted):
		return onJoin(client, chatID, userID, chat)

	case (wasMember || wasAdmin || wasRestricted) && nowLeft:
		return onLeave(client, chatID, userID)

	case nowBanned:
		return onBan(client, chatID, userID)

	case wasBanned && nowLeft:
		slog.Info("User unbanned from chat", "user_id", userID, "chat_id", chatID)
		return nil

	case nowRestricted || wasRestricted:
		return onRestriction(client, chatID, userID, oldStatus, newStatus)

	default:
		return onPromotionOrDemotion(client, chatID, userID, wasAdmin, nowAdmin, chat)
	}
}

// onRestriction handles transitions involving chatMemberStatusRestricted.
func onRestriction(
	client *gotdbot.Client,
	chatID, userID int64,
	oldStatus, newStatus gotdbot.ChatMemberStatus,
) error {
	_, wasRestricted := oldStatus.(*gotdbot.ChatMemberStatusRestricted)
	newRestricted, nowRestricted := newStatus.(*gotdbot.ChatMemberStatusRestricted)

	switch {
	case wasRestricted && !nowRestricted:
		client.Logger.Info("User restriction lifted", "user_id", userID, "chat_id", chatID)

	case !wasRestricted && nowRestricted:
		client.Logger.Info("User restricted in chat", "user_id", userID, "chat_id", chatID)

	default:
		client.Logger.Info("User permissions updated while restricted",
			"user_id", userID,
			"chat_id", chatID,
			"can_send_basic_messages", newRestricted.Permissions.CanSendBasicMessages,
		)
	}

	return nil
}

func onJoin(
	client *gotdbot.Client,
	chatID, userID int64,
	chat *gotdbot.Supergroup,
) error {
	client.Logger.Info("User joined chat", "user_id", userID, "chat_id", chatID)
	if userID == client.Me.Id {
		client.Logger.Info("Bot joined chat", "chat_id", chatID)
		sendJoinLog(client, chatID, chat)
	}

	return nil
}

func onLeave(client *gotdbot.Client, chatID, userID int64) error {
	client.Logger.Info("User left chat", "user_id", userID, "chat_id", chatID)

	if userID == client.Me.Id {
		webapp.StopPlayback(chatID)
	}

	return nil
}

func onBan(client *gotdbot.Client, chatID, userID int64) error {
	client.Logger.Debug("User banned from chat", "user_id", userID, "chat_id", chatID)

	if userID == client.Me.Id {
		webapp.StopPlayback(chatID)
	}

	return nil
}

func onPromotionOrDemotion(
	client *gotdbot.Client,
	chatID, userID int64,
	wasAdmin, nowAdmin bool,
	chat *gotdbot.Supergroup,
) error {
	switch {
	case !wasAdmin && nowAdmin:
		client.Logger.Info("User promoted in chat", "user_id", userID, "chat_id", chatID)

	case wasAdmin && !nowAdmin:
		client.Logger.Info("User demoted in chat", "user_id", userID, "chat_id", chatID)

	default:
		client.Logger.Info("onPromotionOrDemotion", "user_id", userID, "chat_id", chatID)
	}

	return nil
}

// getSupergroup fetches the Supergroup for a chat ID.
// Returns nil (and leaves the chat) when the chat should be abandoned.
func getSupergroup(client *gotdbot.Client, chatID int64) (*gotdbot.Supergroup, error) {
	rawID := stripChannelPrefix(chatID)
	chat, err := client.GetSupergroup(rawID)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid supergroup identifier") {
			_ = client.LeaveChat(chatID)
			return nil, nil
		}

		client.Logger.Error("Failed to fetch supergroup", "chat_id", chatID, "error", err)
		return nil, err
	}

	if chat.IsDirectMessagesGroup {
		_ = client.LeaveChat(chatID)
		return nil, nil
	}

	return chat, nil
}

// stripChannelPrefix converts a full channel ID (e.g. -1001234567890) to its
// bare supergroup ID (1234567890) as expected by GetSupergroup.
func stripChannelPrefix(chatID int64) int64 {
	s := strings.TrimPrefix(strconv.FormatInt(chatID, 10), "-100")
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func sendJoinLog(client *gotdbot.Client, chatID int64, _ *gotdbot.Supergroup) {
	text := fmt.Sprintf("<b>🤖 Bot Joined a New Chat</b>\n📌 <b>Chat ID:</b> <code>%d</code>", chatID)
	if _, err := client.SendTextMessage(config.LoggerId, text, &gotdbot.SendTextMessageOpts{
		ParseMode: "HTML",
	}); err != nil {
		client.Logger.Warn("Failed to send join log", "error", err)
	}
}

// storeChatToDB persists the chat ID in the database
func storeChatToDB(chatID int64) {
	slog.Debug("Storing chat reference", "chat_id", chatID)
	switch {
	case chatID > 0:
		if err := db.Instance.AddUser(chatID); err != nil {
			slog.Error("Failed to store user reference", "chat_id", chatID, "error", err)
		}

	case chatID < 0:
		if err := db.Instance.AddChat(chatID); err != nil {
			slog.Error("Failed to add chat to database", "chat_id", chatID, "error", err)
		}

	default:
		slog.Warn("Invalid chat ID", "chat_id", chatID)
	}
}

// isAdmin reports whether a ChatMemberStatus is admin-level.
func isAdmin(status gotdbot.ChatMemberStatus) bool {
	switch status.(type) {
	case *gotdbot.ChatMemberStatusAdministrator, *gotdbot.ChatMemberStatusCreator:
		return true
	default:
		return false
	}
}

func isStatus[T gotdbot.ChatMemberStatus](status gotdbot.ChatMemberStatus) bool {
	_, ok := status.(T)
	return ok
}
