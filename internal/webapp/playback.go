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
	"ashokshau/tg-web/internal/messaging"
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
func PlayNext(bot *td.Client, chatID int64, quiet ...bool) error {
	return PlayNextForTrack(bot, chatID, "", quiet...)
}

// PlayNextForTrack plays the next song in the queue if the event matches fromTrackID.
func PlayNextForTrack(bot *td.Client, chatID int64, fromTrackID string, quiet ...bool) error {
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

	// Natural transitions inherit the explicit silent intent of the track.
	// A command such as /skip keeps its own per-operation choice.
	if fromTrackID != "" {
		current := cache.ChatCache.GetPlayingTrack(chatID)
		if current != nil && current.Silent {
			quiet = []bool{true}
		}
	}
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
			return PlayTrack(bot, chatID, currentsSong, quiet...)
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
		return PlayTrack(bot, chatID, nextSong, quiet...)
	}

	lastTrack := cache.ChatCache.GetLastAutoplayTrackID(chatID)
	isAutoplay := cache.ChatCache.GetAutoplay(chatID)
	room.mu.Unlock()

	if lastTrack != "" && isAutoplay {
		return handleAutoplay(bot, chatID, lastTrack, quiet...)
	}

	return handleNoSong(bot, chatID, quiet...)
}

func handleAutoplay(bot *td.Client, chatID int64, lastTrack string, quiet ...bool) error {
	history := cache.ChatCache.GetAutoplayHistory(chatID)
	if len(history) >= int(config.AutoPlayLimit) {
		cache.ChatCache.ClearAutoplayHistory(chatID)
		return handleNoSong(bot, chatID, quiet...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit := int(config.AutoPlayLimit)
	tracks, err := downloader.GetRecommendations(ctx, lastTrack, limit)
	if err != nil {
		_, _ = bot.SendMessage(chatID, "Unable to load recommended tracks. Try /play to add another song.", nil)
		return handleNoSong(bot, chatID, true)
	}
	if len(tracks) == 0 {
		return handleNoSong(bot, chatID, quiet...)
	}

	var candidates []utils.GetUrlTrack
	for _, track := range tracks {
		if track.Id == lastTrack || track.Url == lastTrack ||
			slices.Contains(history, track.Id) || slices.Contains(history, track.Url) {
			continue
		}

		candidates = append(candidates, track)
	}

	if len(candidates) == 0 {
		return handleNoSong(bot, chatID, quiet...)
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	nextTrack := candidates[0]
	trackKey := nextTrack.Url
	if trackKey == "" {
		trackKey = nextTrack.Id
	}
	cache.ChatCache.AddAutoplayHistory(chatID, lastTrack, trackKey)

	platform := nextTrack.Platform
	if platform == "" {
		platform = utils.YouTube
	}

	saveCache := &utils.PlayerCache{
		Silent:    isQuiet(quiet),
		URL:       nextTrack.Url,
		Name:      nextTrack.Title,
		User:      utils.AutoPlay,
		Thumbnail: nextTrack.Thumbnail,
		TrackID:   nextTrack.Id,
		Duration:  nextTrack.Duration,
		Channel:   nextTrack.Channel,
		Views:     nextTrack.Views,
		Platform:  platform,
	}

	cache.ChatCache.AddSong(chatID, saveCache)

	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	room.currentTrackID = saveCache.TrackID
	room.Status = "playing"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.mu.Unlock()

	return PlayTrack(bot, chatID, saveCache, quiet...)
}

func StopPlayback(c *td.Client, chatID int64) {
	cache.ChatCache.ClearChat(chatID)
	Manager.Stop(c, chatID)
}

func handleNoSong(bot *td.Client, chatID int64, quiet ...bool) error {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	alreadyStopped := room.Status == "stopped"
	room.Status = "stopped"
	room.currentTrackID = ""
	room.isTransitioning = false
	room.mu.Unlock()

	StopPlayback(bot, chatID)

	if !alreadyStopped {
		_, _ = messaging.Send(bot, chatID, "🎵 Queue finished. Add more songs with /play.", nil, isQuiet(quiet))
	}
	return nil
}

func PlayTrack(bot *td.Client, chatID int64, song *utils.PlayerCache, quiet ...bool) error {
	if song == nil {
		return nil
	}
	reply, err := messaging.Send(bot, chatID, fmt.Sprintf("Downloading %s...", song.Name), nil, song.Silent || isQuiet(quiet))
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
		if reply != nil && reply.Id != 0 {
			_ = reply.Delete(bot, true)
		}
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
			if reply.Id != 0 {
				_, _ = reply.EditText(bot, "⚠️ Download failed. Skipping track...", nil)
			} else {
				_, _ = bot.SendMessage(reply.ChatId, "⚠️ Download failed. Skipping track...", nil)
			}
			room.mu.Lock()
			room.isTransitioning = false
			room.mu.Unlock()
			return PlayNextForTrack(bot, chatID, song.TrackID, reply.Id == 0)
		}
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	if !cache.ChatCache.UpdatePlayingMedia(chatID, song.TrackID, song.FilePath, song.Duration) {
		if reply != nil && reply.Id != 0 {
			_ = reply.Delete(bot, true)
		}
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
	if reply == nil || reply.Id == 0 {
		return nil
	}
	markup := WebAppControlButtons("play", bot.Me.Usernames.EditableUsername, chatID)

	if _, err := messaging.FinishProgress(bot, reply, text, &td.EditMessageTextOpts{
		ReplyMarkup:           markup,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}, song.Silent); err != nil {
		bot.Logger.Error("Failed to update playback message", "chatID", chatID, "error", err)
		return err
	}

	return nil
}

func isQuiet(values []bool) bool { return len(values) > 0 && values[0] }
