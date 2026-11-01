package bot

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/downloader"
	"ashokshau/tg-web/internal/utils"
	"ashokshau/tg-web/internal/webapp"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var inlineTrackCache = cache.NewCache[utils.GetUrlTrack](30 * time.Minute)

func getBotUsername(c *td.Client) string {
	bot := c.Me()
	if bot == nil || bot.Usernames == nil {
		return ""
	}
	return bot.Usernames.EditableUsername
}

func getWebAppURL(botUsername string, userID int64) string {
	return fmt.Sprintf("https://t.me/%s/web?startapp=%d", botUsername, userID)
}

func cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	runes := []rune(title)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return title
}

func richQueueButtons(trackID string, userID int64, webAppURL string) string {
	return fmt.Sprintf(
		"<tg-button-row><tg-button type='callback_data' data='inline_pnow:%s:%d' style='danger'>Play Now</tg-button><tg-button type='callback_data' data='inline_skip:%d:%s'>Skip</tg-button></tg-button-row><tg-button-row><tg-button type='url' url='%s' style='primary'>Join Room</tg-button></tg-button-row>",
		trackID,
		userID,
		userID,
		trackID,
		html.EscapeString(webAppURL),
	)
}

func richJoinRoomButton(webAppURL string) string {
	return fmt.Sprintf("<tg-button-row><tg-button type='url' url='%s' style='primary'>Join Room</tg-button></tg-button-row>", html.EscapeString(webAppURL))
}

func richStartBotButton(botUsername string) string {
	return fmt.Sprintf("<tg-button-row><tg-button type='url' url='https://t.me/%s?start=true' style='primary'>Start Bot</tg-button></tg-button-row>", botUsername)
}

func joinRoomMarkup(webAppURL string) *td.ReplyMarkupInlineKeyboard {
	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "Join Room",
					Type: &td.InlineKeyboardButtonTypeUrl{
						Url: webAppURL,
					},
					Style: td.ButtonStylePrimary{},
				},
			},
		},
	}
}

func buildNowStreamingRich(title, trackURL, duration, webAppURL string) string {
	if trackURL == "" {
		trackURL = webAppURL
	}
	return fmt.Sprintf(
		"🎬 <b>Now Streaming</b><br/><a href='%s'>%s</a><br/><br/><b>Duration:</b> %s%s",
		html.EscapeString(trackURL),
		html.EscapeString(cleanTitle(title)),
		duration,
		richJoinRoomButton(webAppURL),
	)
}

func buildQueueRich(queuePos int, title, trackURL, duration, trackID string, userID int64, webAppURL string) string {
	if trackURL == "" {
		trackURL = webAppURL
	}
	return fmt.Sprintf(
		"🎬 <b>Added to Queue: #%d</b><br/><a href='%s'>%s</a><br/><br/><b>Duration:</b> %s%s",
		queuePos,
		html.EscapeString(trackURL),
		html.EscapeString(cleanTitle(title)),
		duration,
		richQueueButtons(trackID, userID, webAppURL),
	)
}

func buildStartBotRich(botUsername string) string {
	return fmt.Sprintf(
		"🎬 <b>Web Player</b><br/><br/>⚠️ <b>Bot Not Started</b><br/>Please start <a href='https://t.me/%s?start=true'>@%s</a> in private chat to stream music and join rooms.%s",
		botUsername,
		botUsername,
		richStartBotButton(botUsername),
	)
}

func editInlineRichHTMLSafe(c *td.Client, inlineMsgID, richText string) error {
	if inlineMsgID == "" {
		return nil
	}
	err := c.EditInlineRichMessageHTML(inlineMsgID, richText, nil)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not modified") {
		return err
	}
	return nil
}

func inlineQueryHandler(c *td.Client, u *td.UpdateNewInlineQuery) error {
	if u == nil {
		return nil
	}

	botUsername := getBotUsername(c)
	if botUsername == "" {
		return nil
	}

	query := strings.TrimSpace(u.Query)
	if query == "" {
		promptRich := fmt.Sprintf("Type a song name after <code>@%s</code> to search and play in Web Player.", botUsername)
		return u.Answer(c, []td.InputInlineQueryResult{
			&td.InputInlineQueryResultArticle{
				Id:          "search_prompt",
				Title:       "Search for a song",
				Description: "Type a song name or URL to play in Web Player.",
				InputMessageContent: &td.InputMessageRichMessage{
					Message: &td.InputRichMessage{
						Source: &td.RichMessageSourceHtml{Text: promptRich},
					},
				},
			},
		}, &td.AnswerInlineQueryOpts{IsPersonal: true, CacheTime: 1})
	}

	wrapper := downloader.NewDlWrapper(query)
	searchResult, err := wrapper.Search()
	if err != nil || searchResult == nil || len(searchResult.Results) == 0 {
		noResultsRich := fmt.Sprintf("No tracks found for: <code>%s</code>", html.EscapeString(query))
		return u.Answer(c, []td.InputInlineQueryResult{
			&td.InputInlineQueryResultArticle{
				Id:          "no_results",
				Title:       "No results found",
				Description: "No tracks found matching your query.",
				InputMessageContent: &td.InputMessageRichMessage{
					Message: &td.InputRichMessage{
						Source: &td.RichMessageSourceHtml{Text: noResultsRich},
					},
				},
			},
		}, &td.AnswerInlineQueryOpts{IsPersonal: true, CacheTime: 1})
	}

	webAppURL := getWebAppURL(botUsername, u.SenderUserId)

	var results []td.InputInlineQueryResult
	for _, track := range searchResult.Results {
		if len(results) >= 15 {
			break
		}

		if track.Id == "" || track.Duration <= 0 {
			continue
		}
		if config.SongDurationLimit > 0 && track.Duration > config.SongDurationLimit {
			continue
		}

		inlineTrackCache.Set(track.Id, track)

		richContent := buildNowStreamingRich(track.Title, track.Url, utils.SecToMin(track.Duration), webAppURL)

		results = append(results, &td.InputInlineQueryResultArticle{
			Id:           track.Id,
			Title:        track.Title,
			Description:  fmt.Sprintf("%s • %s min", track.Channel, utils.SecToMin(track.Duration)),
			ThumbnailUrl: track.Thumbnail,
			InputMessageContent: &td.InputMessageRichMessage{
				Message: &td.InputRichMessage{
					Source: &td.RichMessageSourceHtml{Text: richContent},
				},
			},
			ReplyMarkup: joinRoomMarkup(webAppURL),
		})
	}

	return u.Answer(c, results, &td.AnswerInlineQueryOpts{IsPersonal: true, CacheTime: 1})
}

func chosenInlineResultHandler(c *td.Client, u *td.UpdateNewChosenInlineResult) error {
	if u == nil {
		return nil
	}

	c.Logger.Debug("Chosen inline result received", "userId", u.SenderUserId, "resultId", u.ResultId, "inlineMessageId", u.InlineMessageId)
	if u.ResultId == "" || u.SenderUserId == 0 {
		return nil
	}

	return handleInlinePlay(c, u.InlineMessageId, u.SenderUserId, u.ResultId)
}

func inlineCallbackHandler(c *td.Client, cb *td.UpdateNewInlineCallbackQuery) error {
	if cb == nil {
		return nil
	}

	c.Logger.Debug("Inline callback query received", "userId", cb.SenderUserId, "data", cb.DataString())
	if cb.SenderUserId == 0 {
		return nil
	}
	data := cb.DataString()

	switch {
	case strings.HasPrefix(data, "inline_pnow:"):
		parts := strings.Split(data, ":")
		if len(parts) < 3 {
			return nil
		}

		trackID := parts[1]
		ownerID, _ := strconv.ParseInt(parts[2], 10, 64)
		if ownerID == 0 {
			ownerID = cb.SenderUserId
		}

		if cb.SenderUserId != ownerID {
			_ = cb.Answer(c, "Only the room owner can control playback.", &td.CallbackAnswerOpts{ShowAlert: true})
			return nil
		}

		botUsername := getBotUsername(c)
		webAppURL := getWebAppURL(botUsername, ownerID)

		if current := cache.ChatCache.GetPlayingTrack(ownerID); current != nil && current.TrackID == trackID {
			_ = cb.Answer(c, "This track is already playing.", nil)
			return nil
		}

		moved := false
		var matchedTrack *utils.PlayerCache
		for _, track := range cache.ChatCache.GetQueue(ownerID) {
			if track != nil && track.TrackID == trackID {
				moved = cache.ChatCache.MoveTrackToFront(ownerID, track.TrackID)
				matchedTrack = track
				break
			}
		}

		if !moved {
			if current := cache.ChatCache.GetPlayingTrack(ownerID); current != nil && current.TrackID == trackID {
				_ = cb.Answer(c, "This track is already playing.", nil)
				return nil
			}

			_ = cb.Answer(c, "Track is preparing or not in queue.", nil)
			return nil
		}

		_ = cb.Answer(c, "Playing now...", nil)
		go func() {
			if err := webapp.PlayNext(c, ownerID); err != nil {
				c.Logger.Warn("Failed to play track now", "error", err)
				return
			}

			if cb.InlineMessageId != "" && matchedTrack != nil {
				newRich := buildNowStreamingRich(matchedTrack.Name, matchedTrack.URL, utils.SecToMin(matchedTrack.Duration), webAppURL)
				_ = editInlineRichHTMLSafe(c, cb.InlineMessageId, newRich)
			}
		}()
		return nil

	case strings.HasPrefix(data, "inline_skip:"):
		parts := strings.Split(data, ":")
		if len(parts) < 2 {
			return nil
		}

		ownerID, _ := strconv.ParseInt(parts[1], 10, 64)
		if ownerID == 0 {
			ownerID = cb.SenderUserId
		}

		if cb.SenderUserId != ownerID {
			_ = cb.Answer(c, "Only the room owner can control playback.", &td.CallbackAnswerOpts{ShowAlert: true})
			return nil
		}

		_ = cb.Answer(c, "Skipping...", nil)
		go func() {
			if err := webapp.PlayNext(c, ownerID); err != nil {
				c.Logger.Warn("Failed to skip track", "error", err)
				return
			}

			if cb.InlineMessageId != "" {
				if nowPlaying := cache.ChatCache.GetPlayingTrack(ownerID); nowPlaying != nil {
					botUsername := getBotUsername(c)
					webAppURL := getWebAppURL(botUsername, ownerID)
					newRich := buildNowStreamingRich(nowPlaying.Name, nowPlaying.URL, utils.SecToMin(nowPlaying.Duration), webAppURL)
					_ = editInlineRichHTMLSafe(c, cb.InlineMessageId, newRich)
				}
			}
		}()
		return nil

	case strings.HasPrefix(data, "inline_play:"):
		trackID := strings.TrimPrefix(data, "inline_play:")
		if idx := strings.Index(trackID, ":"); idx != -1 {
			trackID = trackID[idx+1:]
		}

		if err := cb.Answer(c, "Starting playback...", nil); err != nil {
			c.Logger.Warn("Failed to answer callback query", "error", err)
		}

		return handleInlinePlay(c, cb.InlineMessageId, cb.SenderUserId, trackID)
	}

	return nil
}

func handleInlinePlay(c *td.Client, inlineMsgID string, userID int64, trackID string) error {
	botUsername := getBotUsername(c)
	if botUsername == "" {
		return nil
	}
	webAppURL := getWebAppURL(botUsername, userID)

	c.Logger.Debug("handleInlinePlay started", "userId", userID, "trackID", trackID, "inlineMsgID", inlineMsgID)
	targetChatID := userID
	updater, err := c.SendMessage(targetChatID, "🔍 Searching and downloading...", &td.SendMessageOpts{
		DisableWebPagePreview: true,
	})

	if err != nil {
		c.Logger.Warn("Failed to send PM to user", "userId", userID, "error", err)
		if inlineMsgID != "" {
			startBotRich := buildStartBotRich(botUsername)
			if editErr := editInlineRichHTMLSafe(c, inlineMsgID, startBotRich); editErr != nil {
				c.Logger.Warn("Failed to edit inline start bot message", "error", editErr, "inlineMsgID", inlineMsgID)
			}
		} else {
			c.Logger.Warn("Cannot edit inline start bot message: inlineMsgID is empty", "userId", userID)
		}

		return nil
	}

	if inlineMsgID != "" {
		_ = c.EditInlineMessageReplyMarkup(inlineMsgID, nil)
	}

	go func() {
		track, ok := inlineTrackCache.Get(trackID)
		if !ok {
			wrapper := downloader.NewDlWrapper(trackID)
			if info, err := wrapper.GetInfo(); err == nil && len(info.Results) > 0 {
				track = info.Results[0]
			} else if searchRes, err := wrapper.Search(); err == nil && len(searchRes.Results) > 0 {
				track = searchRes.Results[0]
			} else {
				if _, editErr := updater.EditText(c, "Track not found.", nil); editErr != nil {
					c.Logger.Warn("Failed to edit PM text", "error", editErr)
				}

				if inlineMsgID != "" {
					if editErr := editInlineRichHTMLSafe(c, inlineMsgID, "⚠️ <b>Track not found.</b>"); editErr != nil {
						c.Logger.Warn("Failed to edit inline message", "error", editErr)
					}
					_ = c.EditInlineMessageReplyMarkup(inlineMsgID, nil)
				}
				return
			}
		}

		if config.SongDurationLimit > 0 && track.Duration > config.SongDurationLimit {
			if _, editErr := updater.EditText(c, fmt.Sprintf("Sorry, track exceeds max duration of %d minutes.", config.SongDurationLimit/60), nil); editErr != nil {
				c.Logger.Warn("Failed to edit PM text", "error", editErr)
			}
			if inlineMsgID != "" {
				limitRich := fmt.Sprintf("⚠️ <b>Track exceeds max duration of %d minutes.</b>%s", config.SongDurationLimit/60, richJoinRoomButton(webAppURL))
				if editErr := editInlineRichHTMLSafe(c, inlineMsgID, limitRich); editErr != nil {
					c.Logger.Warn("Failed to edit inline message", "error", editErr)
				}
				_ = c.EditInlineMessageReplyMarkup(inlineMsgID, nil)
			}
			return
		}

		if queueLen := cache.ChatCache.GetQueueLength(userID); queueLen >= 10 {
			if _, editErr := updater.EditText(c, "Queue is full (max 10 tracks). Use '/end' to clear.", nil); editErr != nil {
				c.Logger.Warn("Failed to edit PM text", "error", editErr)
			}

			if inlineMsgID != "" {
				fullRich := fmt.Sprintf("⚠️ <b>Queue is full (max 10 tracks).</b><br/>Use '/end' to clear.%s", richJoinRoomButton(webAppURL))
				if editErr := editInlineRichHTMLSafe(c, inlineMsgID, fullRich); editErr != nil {
					c.Logger.Warn("Failed to edit inline message", "error", editErr)
				}
				_ = c.EditInlineMessageReplyMarkup(inlineMsgID, nil)
			}
			return
		}

		if _track := cache.ChatCache.GetTrackIfExists(userID, track.Id); _track != nil {
			if _, editErr := updater.EditText(c, "Track already in queue or playing.", nil); editErr != nil {
				c.Logger.Warn("Failed to edit PM text", "error", editErr)
			}
			if inlineMsgID != "" {
				existRich := fmt.Sprintf("⚠️ <b>Track already in queue or playing.</b>%s", richJoinRoomButton(webAppURL))
				if editErr := editInlineRichHTMLSafe(c, inlineMsgID, existRich); editErr != nil {
					c.Logger.Warn("Failed to edit inline message", "error", editErr)
				}
				_ = c.EditInlineMessageReplyMarkup(inlineMsgID, nil)
			}
			return
		}

		userMsg := &td.Message{
			ChatId:   userID,
			SenderId: &td.MessageSenderUser{UserId: userID},
		}

		if err = handleSingleTrack(c, userMsg, updater, track, "", userID, false); err != nil {
			c.Logger.Warn("Failed to play inline track", "error", err)
			if inlineMsgID != "" {
				if editErr := editInlineRichHTMLSafe(c, inlineMsgID, "⚠️ <b>Failed to play track.</b>"); editErr != nil {
					c.Logger.Warn("Failed to edit inline message", "error", editErr)
				}
				_ = c.EditInlineMessageReplyMarkup(inlineMsgID, nil)
			}
			return
		}

		if inlineMsgID == "" {
			return
		}

		qLen := cache.ChatCache.GetQueueLength(userID)
		if qLen > 1 {
			queueRich := buildQueueRich(qLen, track.Title, track.Url, utils.SecToMin(track.Duration), track.Id, userID, webAppURL)
			if editErr := editInlineRichHTMLSafe(c, inlineMsgID, queueRich); editErr != nil {
				c.Logger.Warn("Failed to edit inline queue message", "error", editErr)
			}
		}
	}()

	return nil
}
