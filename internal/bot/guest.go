/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/telegram-web-player
 */

package bot

import (
	"errors"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func guestHandler(c *td.Client, u *td.UpdateNewGuestQuery) error {
	if u == nil || u.Message == nil {
		return nil
	}

	if c.Me() == nil || c.Me().Usernames == nil {
		return errors.New("cannot answer guest query: bot username is unavailable")
	}

	username := c.Me().Usernames.EditableUsername

	botURL := "https://t.me/" + username
	message := fmt.Sprintf(`<h3>Music, together on Telegram</h3>
<p>Listen together with <b>Telegram Web Player</b>: synchronized music playback, voice chat, shared queues, and personal playlists.</p>
<p>Find music from YouTube, Spotify, SoundCloud, and other supported platforms.</p>
<p><b>Get started:</b> Add me to your group, use <code>/play [song]</code> to queue music, then open the Web Player with <code>/player</code>.</p>
<p>Open a private chat with me for <code>/help</code> and <code>/privacy</code>.</p>
<tg-button-row align="left"><tg-button type="url" url="%s?startgroup=true">Add to group</tg-button><tg-button type="url" url="%s?start=guest">Open bot</tg-button></tg-button-row>`, botURL, botURL)

	_, err := u.Answer(c, &td.InputInlineQueryResultArticle{
		Id:    "player-introduction",
		Title: "Music, together on Telegram",
		InputMessageContent: &td.InputMessageRichMessage{
			Message: &td.InputRichMessage{
				Source: &td.RichMessageSourceHtml{Text: message},
			},
		},
	})
	return err
}
