/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/messaging"
	"time"

	"github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/filters/callbackquery"
)

var startTime = time.Now()

func LoadModules(c *gotdbot.Client, cancelCleanup func(int64)) {
	register := func(name string, handler func(*gotdbot.Client, *gotdbot.Message) error) {
		c.OnCommand(name, handler)
		if messaging.SupportsSilence(name) {
			c.OnCommand("s"+name, handler)
		}
	}

	register("reload", reloadAdminCacheHandler)
	register("authList", authListHandler)
	register("auths", authListHandler)
	register("auth", addAuthHandler)
	register("addAuth", addAuthHandler)
	register("removeAuth", removeAuthHandler)
	register("rmAuth", removeAuthHandler)
	register("broadcast", broadcastHandler)
	register("gCast", broadcastHandler)
	register("stop_gcast", cancelBroadcastHandler)
	register("stop_broadcast", cancelBroadcastHandler)
	register("av", activeVcHandler)
	register("activevc", activeVcHandler)
	register("logger", loggerHandler)
	register("privacy", privacyHandler)
	register("autoplay", autoplayHandler)
	register("loop", loopHandler)
	register("pause", pauseHandler)
	register("resume", resumeHandler)
	register("cplist", createPlaylistHandler)
	register("createplaylist", createPlaylistHandler)
	register("deleteplaylist", deletePlaylistHandler)
	register("queue", queueHandler)
	register("seek", seekHandler)
	register("sh", shellCommand)
	register("skip", skipHandler)
	register("stop", stopHandler)
	register("end", stopHandler)
	register("start", startHandler)
	register("help", startHandler)
	register("ping", pingHandler)
	register("mix", mixHandler)
	register("play", playHandler)
	register("p", playHandler)
	register("fplay", fPlayHandler)
	register("fp", fPlayHandler)
	register("remove", removeHandler)
	register("settings", settingsHandler)
	register("player", webAppCommandHandler)
	register("addtoplaylist", addToPlaylistHandler)
	register("addtoplist", addToPlaylistHandler)
	register("removefromplaylist", removeFromPlaylistHandler)
	register("rmplist", removeFromPlaylistHandler)
	register("plistinfo", playlistInfoHandler)
	register("playlistinfo", playlistInfoHandler)
	register("myplaylists", myPlaylistsHandler)
	register("myplist", myPlaylistsHandler)
	register("stats", statsHandler)

	c.AddUpdateNewCallbackQueryHandlerGroup(recordTelegramChatContext, nil, -1)
	c.OnUpdateNewCallbackQuery(webAppVerifyCallbackHandler, callbackquery.Prefix("webapp_verify"))
	c.OnUpdateNewCallbackQuery(helpCallbackHandler, callbackquery.Prefix("help_"))
	c.OnUpdateNewCallbackQuery(playCallbackHandler, callbackquery.Prefix("play_"))
	c.OnUpdateNewCallbackQuery(func(client *gotdbot.Client, cb *gotdbot.UpdateNewCallbackQuery) error {
		return settingsCallbackHandler(client, cb, cancelCleanup)
	}, callbackquery.Prefix("settings_"))
	c.OnUpdateNewCallbackQuery(autoplayCallbackHandler, callbackquery.Equal("autoplay_toggle"))

	c.OnUpdateChatMember(handleParticipant, nil)

	c.Logger.Debug("Handlers loaded successfully")
}
