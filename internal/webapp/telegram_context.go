package webapp

import (
	"ashokshau/tg-web/internal/config"
	"errors"
	"strconv"
	"strings"

	tgwebapp "github.com/AshokShau/gotdbot/webapp"
)

type TelegramWebAppContext struct {
	UserID       int64
	Username     string
	ChatType     string
	ChatInstance string
	StartParam   string
	AuthValid    bool
}

func telegramContext(data *tgwebapp.WebAppInitData) TelegramWebAppContext {
	if data == nil || data.User == nil {
		return TelegramWebAppContext{}
	}
	return TelegramWebAppContext{
		UserID: data.User.ID, Username: data.User.Username,
		ChatType: string(data.ChatType), ChatInstance: data.ChatInstance, StartParam: data.StartParam,
		AuthValid: true,
	}
}

func initializeTelegramSession(client *Client, msg ClientMessage, access telegramRoomAccess) bool {
	data, authErr := validateTelegramInitData(msg.InitData, config.Token)
	if authErr != nil {
		log.Warn("[WebApp] Telegram authentication rejected", "room_id", client.RoomID, "reason", authErr.Error())
		sendErrorCode(client, "Valid Telegram authentication required. Reopen the player from Telegram.", "telegram_authentication_required")
		return false
	}

	if msg.RoomID != "" && msg.RoomID != strconv.FormatInt(client.RoomID, 10) {
		sendErrorCode(client, "Room mismatch. Reopen the player for this room.", "signed_room_mismatch")
		return false
	}

	accessErr := validateTelegramRoomAccess(client.RoomID, data, access)
	if accessErr != nil && accessErr.Error() == "private_room_owner_required" && access.invites != nil {
		err := access.invites.checkRoomAccess(client.RoomID, data.User.ID, msg.Password)
		if err != nil {
			switch err.Error() {
			case "partner_password_required":
				sendErrorCode(client, "Enter the owner's passcode to join this player.", "partner_password_required")
				return false
			case "public_access_disabled":
				sendErrorCode(client, "Public access is turned off for this player. Ask the owner to enable public access or set a passcode.", "public_access_disabled")
				return false
			default:
				message := "That passcode didn’t work. Ask your partner to check it."
				if strings.Contains(err.Error(), "Too many attempts") {
					message = err.Error()
				}
				sendErrorCode(client, message, "partner_password_invalid")
				return false
			}
		}
		accessErr = nil
	}
	if err := accessErr; err != nil {
		log.Warn("[WebApp] Room access rejected", "room_id", client.RoomID, "reason", err.Error())
		sendErrorCode(client, telegramRoomAccessMessage(err), err.Error())
		return false
	}

	client.SetUser(data.User.ID, data, data.User.AllowsWriteToPM)
	log.Debug("[WebApp] Telegram session authenticated", "room_id", client.RoomID, "user_id", data.User.ID)
	return true
}

type telegramRoomAccess struct {
	chatInstance func(chatID int64) (string, error)
	invites      *partnerInviteStore
}

func validateTelegramRoomAccess(roomID int64, data *tgwebapp.WebAppInitData, access telegramRoomAccess) error {
	if data == nil || data.User == nil || data.User.ID <= 0 || roomID == 0 {
		return errors.New("invalid_room_identity")
	}

	signedRoomID := data.User.ID
	if data.StartParam != "" {
		var err error
		signedRoomID, err = strconv.ParseInt(data.StartParam, 10, 64)
		if err != nil || signedRoomID == 0 {
			return errors.New("invalid_start_param")
		}
	}

	if signedRoomID != roomID {
		return errors.New("signed_room_mismatch")
	}
	if roomID > 0 {
		if roomID != data.User.ID {
			return errors.New("private_room_owner_required")
		}
		return nil
	}

	if data.ChatType != "group" && data.ChatType != "supergroup" {
		return errors.New("group_launch_required")
	}

	if data.ChatInstance == "" {
		return errors.New("group_context_missing")
	}

	if access.chatInstance == nil {
		return errors.New("chat_binding_unavailable")
	}

	instance, err := access.chatInstance(roomID)
	if err != nil {
		return errors.New("chat_binding_unavailable")
	}

	if instance == "" {
		return errors.New("chat_binding_required")
	}

	if instance != data.ChatInstance {
		return errors.New("group_context_mismatch")
	}
	return nil
}

func telegramRoomAccessMessage(err error) string {
	switch err.Error() {
	case "public_access_disabled":
		return "Public access is turned off for this player. Ask the owner to enable public access or set a passcode."
	case "private_room_owner_required":
		return "This private room belongs to another Telegram user."
	case "group_launch_required", "group_context_mismatch":
		return "Open this player from the linked Telegram group."
	case "group_context_missing":
		return "Open a fresh player button from the linked Telegram group."
	case "chat_binding_required":
		return "Open the player using a fresh bot message in this group. Tap its player button once to verify the chat, then again to open."
	case "chat_binding_unavailable":
		return "Unable to verify access to this Telegram group. Try again."
	default:
		return "Room mismatch. Reopen the player for this room."
	}
}
