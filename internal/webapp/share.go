package webapp

import (
	"ashokshau/tg-web/internal/config"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// prepareShareHandler creates a Telegram inline message for the authenticated user.
// Telegram presents the chat picker and sends it only after the user confirms.
func prepareShareHandler(bot *td.Client, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	initData, valid := verifyTelegramInitData(r.Header.Get("X-Telegram-Init-Data"), config.Token)
	if !valid || initData == nil || initData.User == nil || initData.User.ID <= 0 {
		http.Error(w, "valid Telegram authentication required", http.StatusUnauthorized)
		return
	}
	var track struct {
		Title  string `json:"title"`
		Artist string `json:"artist"`
		URL    string `json:"url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&track); err != nil {
		http.Error(w, "invalid track", http.StatusBadRequest)
		return
	}
	track.Title = strings.Join(strings.Fields(track.Title), " ")
	track.Artist = strings.Join(strings.Fields(track.Artist), " ")
	track.URL = strings.TrimSpace(track.URL)
	link, err := url.Parse(track.URL)
	if track.Title == "" || len([]rune(track.Title)) > 120 || len([]rune(track.Artist)) > 100 || len(track.URL) > 2048 || err != nil || link.User != nil || link.Hostname() == "" || (link.Scheme != "https" && link.Scheme != "http") {
		http.Error(w, "invalid track", http.StatusBadRequest)
		return
	}
	markup := `🎵 <a href="` + html.EscapeString(track.URL) + `">` + html.EscapeString(track.Title) + `</a>`
	if track.Artist != "" {
		markup += "\n" + html.EscapeString(track.Artist)
	}
	formatted, err := bot.ParseText(markup, td.ParseModeHTML)
	if err != nil || formatted == nil {
		http.Error(w, "Telegram could not format this song for sharing", http.StatusBadGateway)
		return
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		http.Error(w, "Telegram could not prepare this song for sharing", http.StatusInternalServerError)
		return
	}
	result := &td.InputInlineQueryResultArticle{
		Id:          "share:" + hex.EncodeToString(nonce[:]),
		Title:       track.Title,
		Description: track.Artist,
		InputMessageContent: &td.InputMessageText{
			Text:               formatted,
			LinkPreviewOptions: &td.LinkPreviewOptions{IsDisabled: true},
		},
		ReplyMarkup: &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{{
			Text:  "Open song",
			Type:  &td.InlineKeyboardButtonTypeUrl{Url: track.URL},
			Style: td.ButtonStylePrimary{},
		}}}},
	}
	prepared, err := bot.SavePreparedInlineMessage(&td.TargetChatTypes{AllowUserChats: true, AllowBotChats: true, AllowGroupChats: true, AllowChannelChats: true}, result, initData.User.ID)
	if err != nil || prepared == nil || prepared.Id == "" {
		http.Error(w, "Telegram could not prepare this song for sharing", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		ID string `json:"id"`
	}{ID: prepared.Id})
}
