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
	"slices"
	"strings"
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
			return PlayTrack(bot, chatID, currentsSong)
		}
	}

	finishedTrack := cache.ChatCache.RemoveCurrentSong(chatID)
	if nextSong := room.nextQueuedOrPrepared(chatID); nextSong != nil {
		room.startNextLocked(nextSong)
		room.mu.Unlock()
		return PlayTrack(bot, chatID, nextSong)
	}

	shouldFetch := finishedTrack != nil && cache.ChatCache.GetAutoplay(chatID) && !room.autoplayUsed
	var fetchContext context.Context
	if shouldFetch {
		room.autoplayUsed = true
		if room.autoplaySeed == nil {
			room.autoplaySeed = finishedTrack
		}
		finishedTrack = room.autoplaySeed
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

func SetAutoplay(bot *td.Client, chatID int64, enabled bool) {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	wasEnabled := cache.ChatCache.GetAutoplay(chatID)
	cache.ChatCache.SetAutoplay(chatID, enabled)
	if enabled && !wasEnabled {
		room.nextTracks = nil
		room.autoplayUsed = false
		room.autoplaySeed = nil
		if playing := cache.ChatCache.GetPlayingTrack(chatID); playing != nil {
			seed := *playing
			room.autoplaySeed = &seed
		}
	} else if !enabled {
		room.autoplaySeed = nil
		if room.autoplayCancel != nil {
			room.autoplayCancel()
			room.autoplayCancel = nil
		}
		room.nextTracks = slices.DeleteFunc(room.nextTracks, func(track *utils.PlayerCache) bool {
			return track == nil || track.User == utils.AutoPlay
		})
	}
	room.mu.Unlock()
}

func SetPlaybackSequence(bot *td.Client, chatID int64, tracks []*utils.PlayerCache) {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	if room.autoplayCancel != nil {
		room.autoplayCancel()
		room.autoplayCancel = nil
	}
	room.isTransitioning = false
	room.transitionID++
	room.nextTracks = tracks
	room.autoplayUsed = false
	room.autoplaySeed = nil
	room.mu.Unlock()
}

type recommendationFlight struct {
	ready  chan struct{}
	tracks []utils.GetUrlTrack
	err    error
}

type recommendationEntry struct {
	limit  int
	tracks []utils.GetUrlTrack
	flight *recommendationFlight
}

func roomRecommendations(ctx context.Context, bot *td.Client, chatID int64, seed string, limit int) ([]utils.GetUrlTrack, error) {
	room := Manager.getOrCreate(bot, chatID)
	room.mu.Lock()
	if entry := room.recommendations[seed]; entry != nil {
		if entry.limit >= limit && len(entry.tracks) > 0 {
			tracks := entry.tracks[:min(limit, len(entry.tracks))]
			room.mu.Unlock()
			return tracks, nil
		}
		if entry.flight != nil {
			flight := entry.flight
			room.mu.Unlock()
			select {
			case <-flight.ready:
				return flight.tracks[:min(limit, len(flight.tracks))], flight.err
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	if room.recommendations == nil {
		room.recommendations = make(map[string]*recommendationEntry)
	}
	if len(room.recommendations) >= 32 {
		for key, entry := range room.recommendations {
			if entry.flight == nil {
				delete(room.recommendations, key)
				break
			}
		}
	}
	flight := &recommendationFlight{ready: make(chan struct{})}
	entry := &recommendationEntry{flight: flight}
	room.recommendations[seed] = entry
	room.mu.Unlock()

	fetchLimit := min(max(limit, int(config.AutoPlayLimit)), maxCollectionTracks)
	tracks, err := downloader.GetRecommendations(ctx, seed, fetchLimit)
	room.mu.Lock()
	flight.tracks, flight.err = tracks, err
	if room.recommendations[seed] == entry {
		if err == nil {
			entry.limit, entry.tracks, entry.flight = fetchLimit, tracks, nil
		} else {
			delete(room.recommendations, seed)
		}
	}
	close(flight.ready)
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
	tracks, err := roomRecommendations(ctx, bot, chatID, seed, min(int(config.AutoPlayLimit), maxCollectionTracks))
	room.mu.Lock()
	if room.transitionID != transitionID || !room.isTransitioning {
		room.mu.Unlock()
		return nil
	}
	if err == nil && cache.ChatCache.GetAutoplay(chatID) {
		seen := make(map[string]bool)
		for _, id := range cache.ChatCache.GetAutoplayHistory(chatID) {
			seen[id] = true
		}
		if finished.URL != "" {
			seen[finished.URL] = true
		}
		seen[finished.TrackID] = true
		for _, track := range tracks {
			if track.Id == "" || track.Title == "" || track.Duration > config.SongDurationLimit || seen[track.Id] || (track.Url != "" && seen[track.Url]) || cache.ChatCache.GetTrackIfExists(chatID, track.Id) != nil {
				continue
			}
			seen[track.Id] = true
			if track.Url != "" {
				seen[track.Url] = true
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
		return PlayTrack(bot, chatID, next)
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

func PlayTrack(bot *td.Client, chatID int64, song *utils.PlayerCache) error {
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
			return PlayNextForTrack(bot, chatID, song.TrackID)
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

	Manager.PlayTrack(bot, chatID, song)
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
