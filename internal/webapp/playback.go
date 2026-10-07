/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/utils"
	"context"
	"fmt"
	"html"
	"log/slog"
	"math/rand"
	"slices"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// PlayNext plays the next song in the queue or handles loop/autoplay.
func PlayNext(bot *td.Client, chatID int64) error {
	return PlayNextForTrack(bot, chatID, "")
}

// PlayNextForTrack plays the next song in the queue if the event matches fromTrackID.
func PlayNextForTrack(bot *td.Client, chatID int64, fromTrackID string) error {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()

	if room.isTransitioning {
		room.mu.Unlock()
		return nil
	}

	if fromTrackID != "" && room.currentTrackID != "" && room.currentTrackID != fromTrackID {
		room.mu.Unlock()
		return nil
	}

	room.isTransitioning = true
	room.cancelTrackEndTimerLocked()

	loop := cache.ChatCache.GetLoopCount(chatID)
	if loop > 0 {
		cache.ChatCache.SetLoopCount(chatID, loop-1)
		if currentsSong := cache.ChatCache.GetPlayingTrack(chatID); currentsSong != nil {
			room.currentTrackID = currentsSong.TrackID
			room.Status = "playing"
			room.Position = 0
			room.ServerTime = time.Now().UnixMilli()
			room.isTransitioning = false
			room.mu.Unlock()
			return PlayTrack(bot, chatID, currentsSong)
		}
	}

	cache.ChatCache.RemoveCurrentSong(chatID)
	if nextSong := cache.ChatCache.GetPlayingTrack(chatID); nextSong != nil {
		room.currentTrackID = nextSong.TrackID
		room.Status = "playing"
		room.Position = 0
		room.ServerTime = time.Now().UnixMilli()
		room.isTransitioning = false
		room.mu.Unlock()
		return PlayTrack(bot, chatID, nextSong)
	}

	lastTrackID := cache.ChatCache.GetLastAutoplayTrackID(chatID)
	isAutoplay := cache.ChatCache.GetAutoplay(chatID)
	room.mu.Unlock()

	if lastTrackID != "" && isAutoplay {
		return handleAutoplay(bot, chatID, lastTrackID)
	}

	return handleNoSong(bot, chatID)
}

func handleAutoplay(bot *td.Client, chatID int64, lastTrackID string) error {
	history := cache.ChatCache.GetAutoplayHistory(chatID)
	if len(history) >= int(config.AutoPlayLimit) {
		cache.ChatCache.ClearAutoplayHistory(chatID)
		return handleNoSong(bot, chatID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tracks, err := downloader.GetYouTubeMixPlaylist(ctx, "RD"+lastTrackID)
	if err != nil || tracks == nil || len(tracks.Results) == 0 {
		return handleNoSong(bot, chatID)
	}

	var candidates []utils.GetUrlTrack
	for _, track := range tracks.Results {
		if track.Id == lastTrackID || slices.Contains(history, track.Id) {
			continue
		}

		candidates = append(candidates, track)
	}

	if len(candidates) == 0 {
		return handleNoSong(bot, chatID)
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	nextTrack := candidates[0]
	cache.ChatCache.AddAutoplayHistory(chatID, lastTrackID, nextTrack.Id)

	saveCache := &utils.PlayerCache{
		URL:       nextTrack.Url,
		Name:      nextTrack.Title,
		User:      utils.AutoPlay,
		Thumbnail: nextTrack.Thumbnail,
		TrackID:   nextTrack.Id,
		Duration:  nextTrack.Duration,
		Channel:   nextTrack.Channel,
		Views:     nextTrack.Views,
		Platform:  utils.YouTube,
	}

	cache.ChatCache.AddSong(chatID, saveCache)

	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	room.currentTrackID = saveCache.TrackID
	room.Status = "playing"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.mu.Unlock()

	return PlayTrack(bot, chatID, saveCache)
}

func StopPlayback(c *td.Client, chatID int64) {
	cache.ChatCache.ClearChat(chatID)
	Manager.Stop(c, chatID)
}

func handleNoSong(bot *td.Client, chatID int64) error {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	alreadyStopped := room.Status == "stopped"
	room.Status = "stopped"
	room.currentTrackID = ""
	room.isTransitioning = false
	room.mu.Unlock()

	StopPlayback(bot, chatID)

	if !alreadyStopped {
		_, _ = bot.SendTextMessage(chatID, "🎵 Queue finished. Add more songs with /play.", nil)
	}
	return nil
}

func PlayTrack(bot *td.Client, chatID int64, song *utils.PlayerCache) error {
	reply, err := bot.SendTextMessage(chatID, fmt.Sprintf("Downloading %s...", song.Name), nil)
	if err != nil {
		slog.Info("[PlayTrack] Failed to send message", "error", err)
		return err
	}

	return PlayTrackWithMessage(bot, reply, chatID, song)
}

func PlayTrackWithMessage(bot *td.Client, reply *td.Message, chatID int64, song *utils.PlayerCache) error {
	if song == nil {
		return nil
	}
	value := *song
	song = &value
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	current := cache.ChatCache.GetPlayingTrack(chatID)
	if current == nil || current.TrackID != song.TrackID {
		room.mu.Unlock()
		return nil
	}
	room.currentTrackID = song.TrackID
	room.cancelTrackEndTimerLocked()
	room.Status = "loading"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.isTransitioning = false
	room.mu.Unlock()
	HubInstance.BroadcastRoomState(bot, chatID)

	if song.FilePath == "" {
		dlPath, err := downloader.DlCachedTrack(song, bot)
		song.FilePath = dlPath
		if err != nil || song.FilePath == "" {
			_, _ = reply.EditText(bot, "⚠️ Download failed. Skipping track...", nil)
			room.mu.Lock()
			room.isTransitioning = false
			room.mu.Unlock()
			return PlayNextForTrack(bot, chatID, song.TrackID)
		}
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	if !cache.ChatCache.UpdatePlayingMedia(chatID, song.TrackID, song.FilePath, song.Duration) {
		return nil
	}

	text := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n"+
			"<b>Title:</b> <a href='%s'>%s</a>\n\n"+
			"<b>Duration:</b> %s min\n"+
			"<b>Requested by:</b> %s",
		html.EscapeString(song.URL),
		html.EscapeString(song.Name),
		utils.SecToMin(song.Duration),
		html.EscapeString(song.User),
	)

	Manager.PlayTrack(bot, chatID, song)
	markup := WebAppControlButtons("play", bot.Me.Usernames.EditableUsername, chatID)

	if _, err := reply.EditText(bot, text, &td.EditTextMessageOpts{
		ReplyMarkup:           markup,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}); err != nil {
		bot.Logger.Error("Failed to update playback message", "chatID", chatID, "error", err)
		return err
	}

	return nil
}
