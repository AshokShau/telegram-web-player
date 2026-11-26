package bot

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/webapp"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func musicActivityTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "a track"
	}
	name := []rune(title)
	if len(name) > 80 {
		title = string(name[:79]) + "…"
	}
	return "“" + title + "”"
}

func publishMusicActivity(c *td.Client, m *td.Message, description string) {
	if m == nil || description == "" {
		return
	}
	userID := m.SenderID()
	isAdmin := m.IsPrivate()
	if !isAdmin && userID > 0 {
		_, err := cache.GetUserAdmin(c, m.ChatId, userID, false)
		isAdmin = err == nil
	}
	webapp.PublishRoomActivity(m.ChatId, userID, firstName(c, m), isAdmin, description)
}

func publishCallbackActivity(c *td.Client, cb *td.UpdateNewCallbackQuery, roomID int64, description string) {
	if cb == nil || description == "" {
		return
	}
	publishUserActivity(c, roomID, cb.SenderUserId, description)
}

func publishUserActivity(c *td.Client, roomID, userID int64, description string) {
	if description == "" {
		return
	}
	name := "Listener"
	if user, err := c.GetUser(userID); err == nil && user != nil && user.FirstName != "" {
		name = user.FirstName
	}
	isAdmin := roomID == userID
	if !isAdmin && userID > 0 {
		_, err := cache.GetUserAdmin(c, roomID, userID, false)
		isAdmin = err == nil
	}
	webapp.PublishRoomActivity(roomID, userID, name, isAdmin, description)
}
