package webapp

import (
	"ashokshau/tg-web/internal/config"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxInviteUsers        = 1000
	maxInviteExpiry       = int64(253402300799)
	maxSearchLength       = 200
	maxPlaylistNameLength = 40
	maxChatMessageLength  = 500
	maxPasscodeLength     = 64
	maxQueueTracks        = 10
	maxPlaylists          = 10
)

func clientLimits() map[string]any {
	return map[string]any{
		"maxTrackDuration": config.SongDurationLimit,
		"maxInviteUsers":   maxInviteUsers, "maxInviteExpiry": maxInviteExpiry,
		"maxSearchLength": maxSearchLength, "maxPlaylistNameLength": maxPlaylistNameLength,
		"maxChatMessageLength": maxChatMessageLength, "maxPasscodeLength": maxPasscodeLength,
		"maxQueueTracks": maxQueueTracks, "maxPlaylists": maxPlaylists,
	}
}

func allowedTrackDuration(seconds int32) bool {
	return seconds >= 0 && seconds <= config.SongDurationLimit
}

func validateInviteSettings(expiresAt int64, userLimit int, now time.Time) error {
	if userLimit < 0 || userLimit > maxInviteUsers {
		return fmt.Errorf("User limit must be a whole number from 0 to %d. Use 0 for unlimited.", maxInviteUsers)
	}
	if expiresAt != 0 && (expiresAt <= now.Unix() || expiresAt > maxInviteExpiry) {
		return errors.New("Choose a valid future expiry date, or leave it empty for no expiry.")
	}
	return nil
}

func validPasscode(value string, allowEmpty bool) bool {
	length := utf8.RuneCountInString(value)
	return allowEmpty && value == "" || length >= 4 && length <= maxPasscodeLength && strings.TrimSpace(value) == value
}

func validateClientMessage(msg ClientMessage) error {
	for _, field := range []struct {
		value string
		limit int
	}{
		{msg.Type, 64}, {msg.RequestID, 64}, {msg.RoomID, 32},
		{msg.PlaylistID, 256}, {msg.TrackID, 256}, {msg.PartnerInvite, 128},
		{msg.SDP, 64 * 1024}, {msg.Candidate, 4096},
	} {
		if len(field.value) > field.limit {
			return errors.New("One of the submitted values is too long.")
		}
	}
	if utf8.RuneCountInString(msg.Password) > maxPasscodeLength {
		return errors.New("Passcodes must contain 4–64 characters.")
	}
	if msg.Track != nil {
		if len(msg.Track.ID) > 256 || len(msg.Track.URL) > 4096 || len(msg.Track.Thumbnail) > 4096 || len(msg.Track.Platform) > 32 || utf8.RuneCountInString(msg.Track.Title) > 512 || utf8.RuneCountInString(msg.Track.Artist) > 512 {
			return errors.New("Track information is too long.")
		}
		if msg.Track.Duration < 0 {
			return errors.New("Track duration must be zero or greater.")
		}
	}
	switch msg.Type {
	case "update_partner_invite":
		return validateInviteSettings(msg.ExpiresAt, msg.UserLimit, time.Now())
	case "set_partner_password":
		if !validPasscode(msg.Password, true) {
			return errors.New("Use a passcode of 4–64 characters without leading or trailing spaces.")
		}
	case "search", "play", "enqueue":
		if utf8.RuneCountInString(strings.TrimSpace(msg.Query)) > maxSearchLength {
			return errors.New("Searches can contain up to 200 characters.")
		}
		if msg.Track != nil && !allowedTrackDuration(msg.Track.Duration) {
			return errors.New("This track exceeds the room’s playback duration limit.")
		}
	case "mix", "recommendations":
		if len(msg.Query) > 4096 || msg.Count < 0 || msg.Count > 25 {
			return errors.New("Choose from 1 to 25 tracks and a valid music link.")
		}
	case "create_playlist", "rename_playlist":
		name := strings.TrimSpace(msg.PlaylistName)
		if name == "" || utf8.RuneCountInString(name) > maxPlaylistNameLength {
			return errors.New("Playlist names must contain 1–40 characters.")
		}
	case "chat_message":
		if utf8.RuneCountInString(strings.TrimSpace(msg.Text)) > maxChatMessageLength {
			return errors.New("Messages can contain up to 500 characters.")
		}
	case "loop":
		if msg.Count < 0 || msg.Count > 10 {
			return errors.New("Repeat count must be from 0 to 10.")
		}
	case "remove":
		if msg.Index < 1 || msg.Index > maxQueueTracks {
			return errors.New("Choose a valid queued track.")
		}
	case "seek":
		if math.IsNaN(msg.PositionSeconds) || math.IsInf(msg.PositionSeconds, 0) || msg.PositionSeconds < 0 {
			return errors.New("Choose a valid playback position.")
		}
	}
	return nil
}

func decodeClientMessage(raw string) (ClientMessage, error) {
	var message ClientMessage
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		return message, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil || fields == nil {
		return message, errors.New("A JSON object is required.")
	}
	for _, key := range []string{"userLimit", "expiresAt", "positionSeconds", "count", "index", "clientTime", "targetUserId", "chatCooldown", "autoDeleteMinutes"} {
		if value, present := fields[key]; present && strings.TrimSpace(string(value)) == "null" {
			return message, errors.New("Numeric fields cannot be null.")
		}
	}
	return message, nil
}

func sendInvalidClientMessage(client *Client, raw string) {
	var envelope struct {
		Type      string `json:"type"`
		RequestID string `json:"requestId"`
	}
	_ = json.Unmarshal([]byte(raw), &envelope)
	if len(envelope.Type) > 64 {
		envelope.Type = ""
	}
	if len(envelope.RequestID) > 64 {
		envelope.RequestID = ""
	}
	payload, _ := json.Marshal(map[string]any{
		"event": "error", "command": envelope.Type, "requestId": envelope.RequestID,
		"data": "Some values are invalid or too large. Check the form and try again.",
	})
	_ = client.SendMessage(string(payload))
}
