/*
 * TgMusicBot - Telegram Music Bot
 * Copyright (c) 2025-2026 Ashok Shau
 *
 * Licensed under GNU GPL v3
 */

package bot

import (
	"fmt"

	"github.com/AshokShau/gotdbot"
)

var privateCommands = []gotdbot.BotCommand{
	{Command: "start", Description: "Open the bot"},
	{Command: "help", Description: "Show commands and help"},
	{Command: "play", Description: "Search for and play a song"},
	{Command: "fplay", Description: "Play a song immediately"},
	{Command: "player", Description: "Open the Web Player"},
	{Command: "queue", Description: "Show the playback queue"},
	{Command: "myplaylists", Description: "Show your playlists"},
	{Command: "privacy", Description: "Read the privacy policy"},
}

var groupAdminCommands = []gotdbot.BotCommand{
	{Command: "play", Description: "Search for and play a song"},
	{Command: "fplay", Description: "Play a song immediately"},
	{Command: "player", Description: "Open the Web Player"},
	{Command: "queue", Description: "Show the playback queue"},
	{Command: "skip", Description: "Skip the current song"},
	{Command: "pause", Description: "Pause playback"},
	{Command: "resume", Description: "Resume playback"},
	{Command: "seek", Description: "Seek to a time in the song"},
	{Command: "stop", Description: "Stop playback"},
	{Command: "remove", Description: "Remove a song from the queue"},
	{Command: "loop", Description: "Set the repeat count"},
	{Command: "settings", Description: "Manage group settings"},
	{Command: "auth", Description: "Authorize a user"},
	{Command: "removeauth", Description: "Remove a user's authorization"},
	{Command: "auths", Description: "Show authorized users"},
	{Command: "reload", Description: "Refresh the administrator cache"},
	{Command: "help", Description: "Show commands and help"},
}

func ConfigureCommandMenus(c *gotdbot.Client) error {
	if err := c.SetCommands(privateCommands, &gotdbot.SetCommandsOpts{
		Scope: &gotdbot.BotCommandScopeAllPrivateChats{},
	}); err != nil {
		return fmt.Errorf("set private commands: %w", err)
	}
	if err := c.DeleteCommands(&gotdbot.DeleteCommandsOpts{
		Scope: &gotdbot.BotCommandScopeAllGroupChats{},
	}); err != nil {
		return fmt.Errorf("clear group commands: %w", err)
	}
	if err := c.DeleteCommands(&gotdbot.DeleteCommandsOpts{
		Scope: &gotdbot.BotCommandScopeAllChatAdministrators{},
	}); err != nil {
		return fmt.Errorf("clear global administrator commands: %w", err)
	}
	if err := c.DeleteCommands(nil); err != nil {
		return fmt.Errorf("clear default commands: %w", err)
	}
	return nil
}

func syncGroupAdminCommands(c *gotdbot.Client, chatID int64, status gotdbot.ChatMemberStatus) error {
	scope := &gotdbot.BotCommandScopeChatAdministrators{ChatId: chatID}
	if isAdmin(status) {
		return c.SetCommands(groupAdminCommands, &gotdbot.SetCommandsOpts{Scope: scope})
	}
	return c.DeleteCommands(&gotdbot.DeleteCommandsOpts{Scope: scope})
}
