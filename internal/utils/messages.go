package utils

import (
	"fmt"
	"html"
	"strings"
)

// TrackTitleHTML leaves Telegram media without a message link as plain text.
func TrackTitleHTML(name, trackURL string) string {
	title := html.EscapeString(name)
	if strings.TrimSpace(trackURL) == "" {
		return title
	}
	return fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(trackURL), title)
}
