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
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/messaging"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"html"
	"slices"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func playHandler(c *td.Client, m *td.Message) error {
	if !playMode(c, m) {
		return td.EndGroups
	}

	return handlePlay(c, m, false)
}

func fPlayHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	return handlePlay(c, m, true)
}

func handlePlay(c *td.Client, m *td.Message, force bool) error {
	chatID := m.ChatId

	if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen > 10 {
		_, _ = m.ReplyText(c, "Queue is full (max 10 tracks). Use /end to clear.", nil)
		return td.EndGroups
	}

	isReply := m.ReplyToMessageID() != 0
	args := Args(m)
	rMsg := m
	var err error
	if isReply {
		rMsg, err = getRepliedMessage(c, m)
		if err != nil {
			_, err = m.ReplyText(c, "Invalid reply message.", nil)
			return err
		}
	}
	url := getUrl(rMsg)
	if isReply && args == "" && url == "" {
		args = rMsg.Text()
	}

	input := coalesce(url, args)

	if strings.HasPrefix(input, "tgpl_") {
		playlist, err := db.Instance.GetPlaylist(input)
		if err != nil {
			_, err = m.ReplyText(c, "❌ Playlist not found.", nil)
			return err
		}

		tracks := db.ConvertSongsToTracks(playlist.Songs)
		if len(tracks) == 0 {
			_, err = m.ReplyText(c, "❌ Playlist is empty.", nil)
			return err
		}

		updater, err := m.ReplyText(c, "🔍 Searching playlist...", nil)
		if err != nil {
			c.Logger.Warn("failed to send message", "error", err)
			return td.EndGroups
		}

		return handleMultipleTracks(c, m, updater, tracks, chatID, force)
	}

	if match := utils.TelegramMessageRegex.FindStringSubmatch(input); match != nil {
		rMsg, err = utils.GetMessage(c, input)
		if err != nil {
			c.Logger.Warn("failed to parse message", "error", err.Error())
			_, err = m.ReplyText(c, "Invalid Telegram link.", nil)
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		if !force && !isReply && !m.IsPrivate() && m.SenderID() > 0 && !cache.ChatCache.IsActive(chatID) {
			if _, adminErr := cache.GetUserAdmin(c, chatID, m.SenderID(), false); adminErr == nil {
				tracks := getProfileTracks(c, m.SenderID())
				if len(tracks) > 0 && !cache.ChatCache.IsActive(chatID) {
					updater, sendErr := m.ReplyText(c, "🔍 Loading profile songs...", nil)
					if sendErr != nil {
						return sendErr
					}
					return handleMultipleTracks(c, m, updater, tracks, chatID, false)
				}
			}
		}
		_, _ = m.ReplyText(c, "<b>Usage:</b>\n/play [song or URL]\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &td.SendMessageOpts{ReplyMarkup: utils.SupportKeyboard(), ParseMode: "HTML"})
		return td.EndGroups
	}

	updater, err := m.ReplyText(c, "🔍 Searching and downloading...", nil)
	if err != nil {
		c.Logger.Warn("failed to send message", "error", err)
		return td.EndGroups
	}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(c, m, updater, rMsg, chatID, force)
	}

	wrapper := downloader.NewDlWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			_, _ = updater.EditText(c, "Invalid URL or unsupported platform.\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &td.EditMessageTextOpts{ReplyMarkup: utils.SupportKeyboard(), ParseMode: "HTML"})
			return td.EndGroups
		}

		trackInfo, err := wrapper.GetInfo()
		if err != nil {
			_, _ = updater.EditText(c, "❌ Error fetching track info: Please try again later.", nil)
			return td.EndGroups
		}

		if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
			_, _ = updater.EditText(c, "No tracks found.", nil)
			return td.EndGroups
		}

		return handleUrl(c, m, updater, trackInfo, chatID, force)
	}

	return handleTextSearch(c, m, updater, wrapper, chatID, force)
}

func handleMedia(c *td.Client, m *td.Message, updater *td.Message, dlMsg *td.Message, chatId int64, force bool) error {
	file, fileName := getFile(dlMsg)
	if file == nil {
		_, err := updater.EditText(c, "No valid media found in the message.", nil)
		return err
	}

	if file.Size > config.MaxFileSize {
		_, err := updater.EditText(c, fmt.Sprintf("File too large. Max size: %d MB.", config.MaxFileSize/(1024*1024)), nil)
		if err != nil {
			c.Logger.Warn("Edit message failed", "error", err)
		}
		return nil
	}

	fileId := dlMsg.RemoteFileID()
	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.EditText(c, "Track already in queue or playing.", nil)
		return err
	}

	dur := dlMsg.RemoteDuration()
	var linkURL string
	link, err := dlMsg.GetLink(c)
	if err != nil {
		c.Logger.Warn("Failed to get file link", "error", err)
	} else if link != nil {
		linkURL = link.Link
	}

	var thumbnail *td.Thumbnail
	var mini *td.Minithumbnail
	var artist string
	switch media := dlMsg.Content.(type) {
	case *td.MessageAudio:
		if media.Audio != nil {
			thumbnail, mini = media.Audio.AlbumCoverThumbnail, media.Audio.AlbumCoverMinithumbnail
			artist = media.Audio.Performer
			if strings.TrimSpace(fileName) == "" {
				fileName = media.Audio.FileName
			}
		}
	case *td.MessageVideo:
		if media.Video != nil {
			thumbnail, mini = media.Video.Thumbnail, media.Video.Minithumbnail
		}
	case *td.MessageVideoNote:
		if media.VideoNote != nil {
			thumbnail, mini = media.VideoNote.Thumbnail, media.VideoNote.Minithumbnail
		}
	case *td.MessageDocument:
		if media.Document != nil {
			thumbnail, mini = media.Document.Thumbnail, media.Document.Minithumbnail
		}
	}

	saveCache := utils.PlayerCache{
		URL: linkURL, Name: fileName, User: firstName(c, m), TrackID: fileId,
		Duration: dur, Platform: utils.Telegram, Channel: artist, Thumbnail: webapp.TelegramThumbnail(thumbnail, mini),
	}

	saveCache.Silent = messaging.SilentCommand(m)
	if queued, err := enqueueTrack(c, updater, chatId, &saveCache, force); queued {
		return err
	}

	file, err = dlMsg.Download(c, 1, &td.DownloadFileOpts{Synchronous: true})
	if err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, "Download failed: Please try again later.", nil)
		return err
	}

	filePath := file.Local.Path
	if dur == 0 {
		dur = utils.GetMediaDuration(filePath)
		saveCache.Duration = dur
	}

	saveCache.FilePath = filePath

	return webapp.PlayTrackWithMessage(c, updater, chatId, &saveCache)
}

func handleTextSearch(c *td.Client, m *td.Message, updater *td.Message, wrapper *downloader.DlWrapper, chatId int64, force bool) error {
	searchResult, err := wrapper.Search()
	if err != nil {
		_, err = updater.EditText(c, "❌ Search failed: Please try again later.", nil)
		return err
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.EditText(c, "😕 No results found. Try a different query.", nil)
		return err
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.Id); _track != nil {
		_, err := updater.EditText(c, "Track already in queue or playing.", nil)
		return err
	}

	return handleSingleTrack(c, m, updater, song, "", chatId, force)
}

func handleUrl(c *td.Client, m *td.Message, updater *td.Message, trackInfo *utils.PlatformTracks, chatId int64, force bool) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.Id); _track != nil {
			_, err := updater.EditText(c, "Track already in queue or playing.", nil)
			return err
		}
		return handleSingleTrack(c, m, updater, track, "", chatId, force)
	}

	return handleMultipleTracks(c, m, updater, trackInfo.Results, chatId, force)
}

func handleSingleTrack(c *td.Client, m *td.Message, updater *td.Message, song utils.GetUrlTrack, filePath string, chatId int64, force bool) error {
	if song.Duration > config.SongDurationLimit {
		_, err := updater.EditText(c, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60), nil)
		return err
	}

	saveCache := utils.PlayerCache{
		URL: song.Url, Name: song.Title, User: firstName(c, m), FilePath: filePath,
		Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration, Channel: song.Channel, Views: song.Views,
		Platform: song.Platform,
	}

	saveCache.Silent = messaging.SilentCommand(m)
	if queued, err := enqueueTrack(c, updater, chatId, &saveCache, force); queued {
		return err
	}

	if saveCache.FilePath == "" {
		dlResult, err := downloader.DlCachedTrack(&saveCache, c)
		if err != nil {
			cache.ChatCache.RemoveCurrentSong(chatId)
			_, err = updater.EditText(c, "Download failed: Please try again later.", nil)
			return err
		}

		saveCache.FilePath = dlResult
	}

	return webapp.PlayTrackWithMessage(c, updater, chatId, &saveCache)
}

func handleMultipleTracks(c *td.Client, m *td.Message, updater *td.Message, tracks []utils.GetUrlTrack, chatId int64, force bool) error {
	if len(tracks) == 0 {
		_, err := updater.EditText(c, "No tracks found.", nil)
		return err
	}

	queueHeader := "<u><b>Added to Queue:</b></u>\n<blockquote expandable>\n"
	var tracksToAdd []*utils.PlayerCache
	var skippedTracks []string

	for _, track := range tracks {
		if track.Duration > config.SongDurationLimit {
			skippedTracks = append(skippedTracks, track.Title)
			continue
		}

		saveCache := &utils.PlayerCache{
			Silent: messaging.SilentCommand(m), Name: track.Title, TrackID: track.Id, Duration: track.Duration,
			Thumbnail: track.Thumbnail, User: firstName(c, m), Platform: track.Platform,
			URL: track.Url, Channel: track.Channel, Views: track.Views,
		}
		tracksToAdd = append(tracksToAdd, saveCache)
	}

	if len(tracksToAdd) == 0 {
		_, err := updater.EditText(c, fmt.Sprintf("All tracks were skipped (max duration %d min).", config.SongDurationLimit/60), nil)
		return err
	}

	if len(skippedTracks) > 0 {
		_, _ = m.ReplyText(c, fmt.Sprintf("Skipped %d tracks because they exceed the duration limit.", len(skippedTracks)), nil)
	}

	var qLenAfter int
	var startLen int

	if force {
		qLenAfter = 0
		for _, t := range slices.Backward(tracksToAdd) {
			qLenAfter = cache.ChatCache.AddSongToFront(chatId, t)
		}
		startLen = qLenAfter - len(tracksToAdd)
		if startLen > 0 {
			if err := webapp.PlayNext(c, chatId, messaging.Quiet(m)); err != nil {
				return err
			}
			if updater.Id != 0 {
				_ = c.DeleteMessages(chatId, []int64{updater.Id}, &td.DeleteMessagesOpts{Revoke: true})
			}
			webapp.HubInstance.BroadcastRoomState(c, chatId)
			return nil
		}
	} else {
		qLenAfter = cache.ChatCache.AddSongs(chatId, tracksToAdd)
		startLen = qLenAfter - len(tracksToAdd)
	}

	if startLen == 0 {
		firstTrack := tracksToAdd[0]
		return webapp.PlayTrackWithMessage(c, updater, chatId, firstTrack)
	}

	var sb strings.Builder
	sb.WriteString(queueHeader)

	var totalDuration int32
	for i, track := range tracksToAdd {
		currentQLen := startLen + i + 1
		escTrackName := html.EscapeString(track.Name)
		fmt.Fprintf(&sb, "<b>%d.</b> %s\n└ Duration: %s\n",
			currentQLen, escTrackName, utils.SecToMin(track.Duration))
		totalDuration += track.Duration
	}

	sb.WriteString("</blockquote>")
	escRequester := html.EscapeString(firstName(c, m))
	queueSummary := fmt.Sprintf(
		"\n<b>Queue Total:</b> %d\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		qLenAfter, utils.SecToMin(totalDuration), escRequester,
	)

	sb.WriteString(queueSummary)

	fullMessage := sb.String()

	if len(fullMessage) > 4096 {
		fullMessage = queueSummary
	}

	webapp.HubInstance.BroadcastRoomState(c, chatId)

	_, err := messaging.FinishProgress(c, updater, fullMessage, &td.EditMessageTextOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           utils.QueueMarkup(tracksToAdd[0].TrackID, c.Me.Usernames.EditableUsername, chatId),
		DisableWebPagePreview: true,
	}, messaging.SilentCommand(m))

	return err
}

func enqueueTrack(c *td.Client, updater *td.Message, chatId int64, saveCache *utils.PlayerCache, force bool) (bool, error) {
	var qLen int
	if force {
		qLen = cache.ChatCache.AddSongToFront(chatId, saveCache)
	} else {
		qLen = cache.ChatCache.AddSong(chatId, saveCache)
	}

	webapp.HubInstance.BroadcastRoomState(c, chatId)

	if qLen > 1 {
		if force {
			if err := webapp.PlayNext(c, chatId, saveCache.Silent); err != nil {
				return true, err
			}
			if updater.Id != 0 {
				_ = c.DeleteMessages(chatId, []int64{updater.Id}, &td.DeleteMessagesOpts{Revoke: true})
			}
			return true, nil
		}
		escUser := html.EscapeString(saveCache.User)
		title := html.EscapeString(saveCache.Name)
		if strings.TrimSpace(saveCache.URL) != "" {
			title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(saveCache.URL), title)
		}
		queueInfo := fmt.Sprintf(
			"<u><b>Added to queue: %d</b></u>\n\n<b>Title:</b> %s\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
			qLen, title, utils.SecToMin(saveCache.Duration), escUser,
		)
		_, err := messaging.FinishProgress(c, updater, queueInfo, &td.EditMessageTextOpts{
			ReplyMarkup:           utils.QueueMarkup(saveCache.TrackID, c.Me.Usernames.EditableUsername, chatId),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		}, saveCache.Silent)
		return true, err
	}

	return false, nil
}
