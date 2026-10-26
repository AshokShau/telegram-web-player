/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()
	if strings.Contains(data, "play_close") {
		_ = cb.Answer(c, "Closing...", &td.CallbackAnswerOpts{ShowAlert: false})
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}

	if !cache.ChatCache.IsActive(chatID) {
		text := "There is no active playback."
		_ = cb.Answer(c, text, &td.CallbackAnswerOpts{ShowAlert: false})
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, "There is no active playback.", nil)
		_, _ = cb.EditMessageText(c, "There is no active playback.", &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		escUser := html.EscapeString(currentTrack.User)
		title := html.EscapeString(currentTrack.Name)
		if strings.TrimSpace(currentTrack.URL) != "" {
			title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(currentTrack.URL), title)
		}
		return fmt.Sprintf("%s <b>%s</b>\n\n<b>Track:</b> %s\n<b>Duration:</b> %s\n<b>Requested by:</b> %s",
			emoji, status,
			title,
			utils.SecToMin(currentTrack.Duration),
			escUser,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err = webapp.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to skip the current track.", nil)
			_, _ = cb.EditMessageText(c, "Unable to skip the current track.", &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		_ = cb.Answer(c, "Track skipped.", nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		webapp.StopPlayback(c, chatID)

		msg := fmt.Sprintf("<b>Playback stopped.</b>\nRequested by: %s", html.EscapeString(user.FirstName))
		_ = cb.Answer(c, "Playback stopped.", nil)
		_, err = cb.EditMessageText(c, msg, &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = webapp.Manager.Pause(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to pause playback.", nil)
			_, _ = cb.EditMessageText(c, "Unable to pause playback.", &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		_ = cb.Answer(c, "Playback paused.", nil)
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\nPaused by %s", html.EscapeString(user.FirstName))
		markup := webapp.WebAppControlButtons("pause", c.Me().Usernames.EditableUsername, chatID)
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err = webapp.Manager.Resume(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to resume playback.", &td.CallbackAnswerOpts{ShowAlert: false})
			_, _ = cb.EditMessageText(c, "Unable to resume playback.", &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("pause", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, "Playback resumed.", &td.CallbackAnswerOpts{ShowAlert: true})
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nResumed by %s", html.EscapeString(user.FirstName))
		markup := webapp.WebAppControlButtons("resume", c.Me().Usernames.EditableUsername, chatID)
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, "Unable to fetch playlists.", nil)
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, "Unable to create playlist.", &td.CallbackAnswerOpts{ShowAlert: true})
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}

		err = db.Instance.AddSongToPlaylist(playlistID, song, cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, "Unable to add track to playlist.", nil)
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, "Playlist not found.", nil)
			return nil
		}

		_ = cb.Answer(c, fmt.Sprintf("Track \"%s\" added to playlist \"%s\".", song.Name, playlist.Name), nil)
		return nil

	case strings.HasPrefix(data, "play_now_"):
		moved := false
		for _, track := range cache.ChatCache.GetQueue(chatID) {
			if track != nil && utils.PlayNowCallbackData(track.TrackID) == data {
				moved = cache.ChatCache.MoveTrackToFront(chatID, track.TrackID)
				break
			}
		}
		if !moved {
			_ = cb.Answer(c, "Track not found in queue.", nil)
			return nil
		}

		if err = webapp.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to play the track.", nil)
			return nil
		}

		_ = cb.Answer(c, "Playing now.", nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	text := buildTrackMessage("Now Playing", "▶")
	_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: webapp.WebAppControlButtons("resume", c.Me().Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
	return nil
}
