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
		_ = cb.Answer(c, 0, false, "Closing...", "")
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
		_ = cb.Answer(c, 0, false, text, "")
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, 0, false, "There is no active playback.", "")
		_, _ = cb.EditMessageText(c, "There is no active playback.", &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		escURL := html.EscapeString(currentTrack.URL)
		escName := html.EscapeString(currentTrack.Name)
		escUser := html.EscapeString(currentTrack.User)
		return fmt.Sprintf("%s <b>%s</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>Requested by:</b> %s",
			emoji, status,
			escURL, escName,
			utils.SecToMin(currentTrack.Duration),
			escUser,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err = webapp.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to skip the current track.", "")
			_, _ = cb.EditMessageText(c, "Unable to skip the current track.", &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		_ = cb.Answer(c, 0, false, "Track skipped.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		webapp.StopPlayback(c, chatID)

		msg := fmt.Sprintf("<b>Playback stopped.</b>\nRequested by: %s", html.EscapeString(user.FirstName))
		_ = cb.Answer(c, 0, false, "Playback stopped.", "")
		_, err = cb.EditMessageText(c, msg, &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = webapp.Manager.Pause(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to pause playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to pause playback.", &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		_ = cb.Answer(c, 0, false, "Playback paused.", "")
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\nPaused by %s", html.EscapeString(user.FirstName))
		markup := webapp.WebAppControlButtons("pause", c.Me.Usernames.EditableUsername, chatID)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err = webapp.Manager.Resume(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to resume playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to resume playback.", &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("pause", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback resumed.", "")
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nResumed by %s", html.EscapeString(user.FirstName))
		markup := webapp.WebAppControlButtons("resume", c.Me.Usernames.EditableUsername, chatID)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Unable to fetch playlists.", "")
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, 0, false, "Unable to create playlist.", "")
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
			_ = cb.Answer(c, 0, false, "Unable to add track to playlist.", "")
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Playlist not found.", "")
			return nil
		}

		_ = cb.Answer(c, 0, false, fmt.Sprintf("Track \"%s\" added to playlist \"%s\".", song.Name, playlist.Name), "")
		return nil

	case strings.HasPrefix(data, "play_now_"):
		trackID := strings.TrimPrefix(data, "play_now_")
		if ok := cache.ChatCache.MoveTrackToFront(chatID, trackID); !ok {
			_ = cb.Answer(c, 0, false, "Track not found in queue.", "")
			return nil
		}

		if err = webapp.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to play the track.", "")
			return nil
		}

		_ = cb.Answer(c, 0, false, "Playing now.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	text := buildTrackMessage("Now Playing", "▶")
	_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: webapp.WebAppControlButtons("resume", c.Me.Usernames.EditableUsername, chatID), ParseMode: "HTML", DisableWebPagePreview: true})
	return nil
}
