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
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// PlayNext plays the next song in the queue or handles loop/autoplay.
func PlayNext(bot *td.Client, chatID int64) error {
	return playNextForTrack(bot, chatID, "")
}

func playNextForTrack(bot *td.Client, chatID int64, fromTrackID string) error {
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
	room.transitionID++
	transitionID := room.transitionID
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
			return playTrack(bot, chatID, currentsSong)
		}
	}

	finishedTrack := cache.ChatCache.RemoveCurrentSong(chatID)
	if nextSong := room.nextQueuedOrPrepared(chatID); nextSong != nil {
		room.startNextLocked(nextSong)
		room.mu.Unlock()
		return playTrack(bot, chatID, nextSong)
	}

	shouldFetch := finishedTrack != nil && cache.ChatCache.GetAutoplay(chatID)
	var fetchContext context.Context
	if shouldFetch {
		var cancel context.CancelFunc
		fetchContext, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		room.autoplayCancel = cancel
	}
	room.mu.Unlock()

	if shouldFetch {
		return prepareAutoplay(bot, chatID, finishedTrack, transitionID, fetchContext)
	}

	return handleNoSong(bot, chatID)
}

func (room *RoomState) nextQueuedOrPrepared(chatID int64) *utils.PlayerCache {
	if next := cache.ChatCache.GetPlayingTrack(chatID); next != nil {
		if cache.ChatCache.GetAutoplay(chatID) && next.User != utils.AutoPlay {
			room.nextTracks = nil
			room.autoplaySeed = nil
		}
		return next
	}
	for len(room.nextTracks) > 0 {
		next := room.nextTracks[0]
		room.nextTracks[0] = nil
		room.nextTracks = room.nextTracks[1:]
		if next == nil || (next.User == utils.AutoPlay && !cache.ChatCache.GetAutoplay(chatID)) {
			continue
		}
		cache.ChatCache.AddSong(chatID, next)
		return next
	}
	return nil
}

func (room *RoomState) selectUpNext(chatID int64, index int, trackID string) bool {
	room.mu.Lock()
	defer room.mu.Unlock()
	queue := cache.ChatCache.GetQueue(chatID)
	if room.isTransitioning || len(queue) == 0 {
		return false
	}
	queued := len(queue) - 1
	if index < 0 || trackID == "" || index >= queued+len(room.nextTracks) {
		return false
	}
	if index < queued {
		return queue[index+1] != nil && queue[index+1].TrackID == trackID && cache.ChatCache.MoveTrackToFront(chatID, trackID)
	}
	index -= queued
	next := room.nextTracks[index]
	if next == nil || next.TrackID != trackID {
		return false
	}
	room.nextTracks = room.nextTracks[index+1:]
	cache.ChatCache.AddSongToFront(chatID, next)
	return true
}

func (room *RoomState) removeUpNext(chatID int64, index int, trackID string) bool {
	room.mu.Lock()
	defer room.mu.Unlock()
	queue := cache.ChatCache.GetQueue(chatID)
	if index < 1 {
		return false
	}
	if index < len(queue) {
		return queue[index] != nil && (trackID == "" || queue[index].TrackID == trackID) && cache.ChatCache.RemoveTrack(chatID, index)
	}
	index -= max(1, len(queue))
	if index >= len(room.nextTracks) || room.nextTracks[index] == nil || (trackID != "" && room.nextTracks[index].TrackID != trackID) {
		return false
	}
	copy(room.nextTracks[index:], room.nextTracks[index+1:])
	room.nextTracks[len(room.nextTracks)-1] = nil
	room.nextTracks = room.nextTracks[:len(room.nextTracks)-1]
	return true
}

func (room *RoomState) clearUpNext(chatID int64) {
	room.mu.Lock()
	defer room.mu.Unlock()
	queue := cache.ChatCache.GetQueue(chatID)
	for i := len(queue) - 1; i >= 1; i-- {
		cache.ChatCache.RemoveTrack(chatID, i)
	}
	room.nextTracks = nil
}

func (room *RoomState) startNextLocked(next *utils.PlayerCache) {
	room.currentTrackID = next.TrackID
	room.Status = "playing"
	room.Position = 0
	room.ServerTime = time.Now().UnixMilli()
	room.isTransitioning = false
}

func recommendationSeed(track *utils.PlayerCache) string {
	if track == nil {
		return ""
	}
	if track.Platform != utils.Telegram && track.URL != "" {
		return track.URL
	}
	return strings.TrimSpace(track.Name + " " + track.Channel)
}

func setAutoplay(bot *td.Client, chatID int64, enabled bool) {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	if cache.ChatCache.GetAutoplay(chatID) == enabled {
		room.mu.Unlock()
		return
	}
	cache.ChatCache.SetAutoplay(chatID, enabled)
	room.nextTracks = nil
	room.autoplaySeed = nil
	if room.autoplayCancel != nil {
		room.autoplayCancel()
		room.autoplayCancel = nil
	}
	room.transitionID++
	room.isTransitioning = false
	room.mu.Unlock()
}

func SetPlaybackSequence(bot *td.Client, chatID int64, tracks []*utils.PlayerCache, collection bool) {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	if room.autoplayCancel != nil {
		room.autoplayCancel()
		room.autoplayCancel = nil
	}
	room.isTransitioning = false
	room.transitionID++
	if collection {
		cache.ChatCache.SetAutoplay(chatID, false)
	}
	room.nextTracks = tracks
	room.autoplaySeed = nil
	room.mu.Unlock()
}

func roomRecommendations(ctx context.Context, chatID int64, seed string, limit int) ([]utils.GetUrlTrack, error) {
	if tracks := cache.ChatCache.GetRecommendations(chatID, seed, limit); tracks != nil {
		return tracks, nil
	}
	room := Manager.getOrCreate(nil, chatID)
	room.mu.Lock()
	revision := room.transitionID
	room.mu.Unlock()
	tracks, err := downloader.GetRecommendations(ctx, seed, maxRecommendationTracks)
	room.mu.Lock()
	if err == nil && ctx.Err() == nil && room.transitionID == revision {
		cache.ChatCache.SetRecommendations(chatID, seed, tracks)
	}
	room.mu.Unlock()
	return tracks[:min(limit, len(tracks))], err
}

func prepareAutoplay(bot *td.Client, chatID int64, finished *utils.PlayerCache, transitionID uint64, ctx context.Context) error {
	seed := recommendationSeed(finished)
	room := Manager.getOrCreate(bot, chatID)
	defer func() {
		room.mu.Lock()
		if room.transitionID == transitionID && room.autoplayCancel != nil {
			room.autoplayCancel()
			room.autoplayCancel = nil
		}
		room.mu.Unlock()
	}()
	tracks, err := roomRecommendations(ctx, chatID, seed, 10)
	room.mu.Lock()
	if room.transitionID != transitionID || !room.isTransitioning {
		room.mu.Unlock()
		return nil
	}
	if err == nil && cache.ChatCache.GetAutoplay(chatID) {
		room.autoplaySeed = finished
		for _, track := range tracks {
			if track.Id == "" || track.Title == "" || track.Duration > config.SongDurationLimit || track.Id == finished.TrackID || cache.ChatCache.GetTrackIfExists(chatID, track.Id) != nil {
				continue
			}
			platform := track.Platform
			if platform == "" {
				platform = utils.YouTube
			}
			room.nextTracks = append(room.nextTracks, &utils.PlayerCache{URL: track.Url, Name: track.Title, User: utils.AutoPlay, Thumbnail: track.Thumbnail, TrackID: track.Id, Duration: track.Duration, Channel: track.Channel, Views: track.Views, Platform: platform})
		}
	}
	if next := room.nextQueuedOrPrepared(chatID); next != nil {
		room.startNextLocked(next)
		room.mu.Unlock()
		return playTrack(bot, chatID, next)
	}
	room.mu.Unlock()
	return handleNoSong(bot, chatID)
}

func StopPlayback(c *td.Client, chatID int64) {
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

	if !alreadyStopped && bot != nil {
		_, _ = bot.SendMessage(chatID, "🎵 Queue finished. Add more songs with /play.", nil)
	}
	return nil
}

func playTrack(bot *td.Client, chatID int64, song *utils.PlayerCache) error {
	if song == nil {
		return nil
	}
	var reply *td.Message
	var err error
	if bot != nil {
		reply, err = bot.SendMessage(chatID, fmt.Sprintf("Downloading %s...", song.Name), nil)
		if err != nil {
			slog.Info("[PlayTrack] Failed to send message", "error", err)
			return err
		}
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
		if reply != nil && reply.Id != 0 && bot != nil {
			_ = reply.Delete(bot, &td.DeleteMessagesOpts{Revoke: true})
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
			if reply != nil && reply.Id != 0 {
				_, _ = reply.EditText(bot, "⚠️ Download failed. Skipping track...", nil)
			} else if bot != nil {
				_, _ = bot.SendMessage(chatID, "⚠️ Download failed. Skipping track...", nil)
			}
			room.mu.Lock()
			room.isTransitioning = false
			room.mu.Unlock()
			return playNextForTrack(bot, chatID, song.TrackID)
		}
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	if !cache.ChatCache.UpdatePlayingMedia(chatID, song.TrackID, song.FilePath, song.Duration) {
		if reply != nil && reply.Id != 0 && bot != nil {
			_ = reply.Delete(bot, &td.DeleteMessagesOpts{Revoke: true})
		}
		return nil
	}

	title := html.EscapeString(song.Name)
	if strings.TrimSpace(song.URL) != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(song.URL), title)
	}
	text := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n"+
			"<b>Title:</b> %s\n\n"+
			"<b>Duration:</b> %s min\n"+
			"<b>Requested by:</b> %s",
		title,
		utils.SecToMin(song.Duration),
		html.EscapeString(song.User),
	)

	Manager.playTrack(bot, chatID, song)
	if reply == nil || reply.Id == 0 || bot == nil {
		return nil
	}
	markup := WebAppControlButtons("play", bot.Me().Usernames.EditableUsername, chatID)

	if _, err := reply.EditText(bot, text, &td.EditMessageTextOpts{
		ReplyMarkup:           markup,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}); err != nil {
		bot.Logger.Error("Failed to update playback message", "chatID", chatID, "error", err)
		return err
	}

	return nil
}
