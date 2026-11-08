package cache

import (
	"ashokshau/tg-web/internal/utils"
	"testing"
)

func TestChatCacherQueueOperations(t *testing.T) {
	c := newChatCacher()
	chatID := int64(1001)

	if c.IsActive(chatID) {
		t.Fatalf("expected chat to be inactive initially")
	}

	song1 := &utils.PlayerCache{TrackID: "t1", Name: "Song 1", URL: "http://song1.mp3", Duration: 120}
	song2 := &utils.PlayerCache{TrackID: "t2", Name: "Song 2", URL: "http://song2.mp3", Duration: 180}

	length := c.AddSong(chatID, song1)
	if length != 1 {
		t.Fatalf("expected queue length 1, got %d", length)
	}

	if !c.IsActive(chatID) {
		t.Fatalf("expected chat to be active")
	}

	length = c.AddSong(chatID, song2)
	if length != 2 {
		t.Fatalf("expected queue length 2, got %d", length)
	}

	playing := c.GetPlayingTrack(chatID)
	if playing == nil || playing.TrackID != "t1" {
		t.Fatalf("expected playing track to be t1, got %v", playing)
	}

	upcoming := c.GetUpcomingTrack(chatID)
	if upcoming == nil || upcoming.TrackID != "t2" {
		t.Fatalf("expected upcoming track to be t2, got %v", upcoming)
	}

	song3 := &utils.PlayerCache{TrackID: "t3", Name: "Song 3"}
	c.AddSongToFront(chatID, song3)
	// Now queue should be: song1 (playing, idx 0), song3 (idx 1), song2 (idx 2)
	if c.GetUpcomingTrack(chatID).TrackID != "t3" {
		t.Fatalf("expected upcoming track after AddSongToFront to be t3")
	}

	moved := c.MoveTrackToFront(chatID, "t2")
	if !moved {
		t.Fatalf("expected MoveTrackToFront to succeed")
	}
	// Now queue should be: song1 (idx 0), song2 (idx 1), song3 (idx 2)
	if c.GetUpcomingTrack(chatID).TrackID != "t2" {
		t.Fatalf("expected upcoming track after MoveTrackToFront to be t2")
	}

	removed := c.RemoveCurrentSong(chatID)
	if removed == nil || removed.TrackID != "t1" {
		t.Fatalf("expected removed song to be t1")
	}

	// Now queue: song2 (idx 0), song3 (idx 1)
	if c.GetPlayingTrack(chatID).TrackID != "t2" {
		t.Fatalf("expected playing track to be t2")
	}

	c.RemoveTrack(chatID, 0)
	// Now queue: song3 (idx 0)
	if c.GetPlayingTrack(chatID).TrackID != "t3" {
		t.Fatalf("expected playing track to be t3")
	}

	c.ClearChat(chatID)
	if c.IsActive(chatID) {
		t.Fatalf("expected chat to be inactive after ClearChat")
	}
}

func TestChatCacherAutoplayAndLoop(t *testing.T) {
	c := newChatCacher()
	chatID := int64(2002)

	c.SetAutoplay(chatID, true)
	if !c.GetAutoplay(chatID) {
		t.Fatalf("expected autoplay to be true")
	}

	// Test Loop count
	song := &utils.PlayerCache{TrackID: "t1", Loop: 0}
	c.AddSong(chatID, song)

	c.SetLoopCount(chatID, 3)
	if c.GetLoopCount(chatID) != 3 {
		t.Fatalf("expected loop count 3, got %d", c.GetLoopCount(chatID))
	}
}

func TestClearChatRemovesAllStateForOneChat(t *testing.T) {
	c := newChatCacher()
	const chatID int64 = 2003
	const otherChatID int64 = 2004
	c.SetAutoplay(chatID, true)
	c.AddSong(chatID, &utils.PlayerCache{TrackID: "song"})
	c.RemoveCurrentSong(chatID)
	c.AddSong(chatID, &utils.PlayerCache{TrackID: "queued"})
	c.AddChatMessage(chatID, &ChatMessage{Text: "hello"})
	c.CheckAndSetCooldown(chatID, 42, true, 30)
	c.SetAutoplay(otherChatID, true)

	c.ClearChat(chatID)
	if _, exists := c.chatCache[chatID]; exists || c.GetAutoplay(chatID) || c.IsActive(chatID) || len(c.GetChatHistory(chatID)) != 0 {
		t.Fatal("chat state remained after clear")
	}
	if !c.GetAutoplay(otherChatID) {
		t.Fatal("clearing one chat changed another chat")
	}
}

func TestTrackSnapshotsAndStaleDownload(t *testing.T) {
	c := newChatCacher()
	original := &utils.PlayerCache{TrackID: "first", Name: "First", Duration: 60}
	c.AddSong(42, original)
	original.FilePath = "outside-mutation"
	if c.GetPlayingTrack(42).FilePath != "" {
		t.Fatal("caller mutated the cached track")
	}
	snapshot := c.GetPlayingTrack(42)
	snapshot.Name = "changed"
	queue := c.GetQueue(42)
	queue[0].Name = "changed again"
	if c.GetPlayingTrack(42).Name != "First" {
		t.Fatal("snapshot exposed mutable cache memory")
	}
	if !c.UpdatePlayingMedia(42, "first", "first.mp3", 90) {
		t.Fatal("current download was not published")
	}
	if c.GetPlayingTrack(42).Duration != 90 {
		t.Fatal("duration was not published")
	}
	c.AddSong(42, &utils.PlayerCache{TrackID: "second"})
	c.RemoveCurrentSong(42)
	if c.UpdatePlayingMedia(42, "first", "late.mp3", 90) {
		t.Fatal("stale download overwrote the new current track")
	}
	if c.GetPlayingTrack(42).FilePath != "" {
		t.Fatal("stale download modified the next track")
	}
	c.ClearChat(42)
	if c.UpdatePlayingMedia(42, "second", "late.mp3", 90) {
		t.Fatal("download resurrected stopped playback")
	}
}

func TestRecommendationsAreSharedAndCleared(t *testing.T) {
	c := newChatCacher()
	chatID := int64(3003)
	tracks := []utils.GetUrlTrack{{Id: "first", Title: "First"}, {Id: "second", Title: "Second"}}
	c.SetRecommendations(chatID, "seed", tracks)
	tracks[0].Title = "Changed"
	got := c.GetRecommendations(chatID, "seed", 1)
	if len(got) != 1 || got[0].Title != "First" || len(c.GetRecommendations(chatID, "seed", 25)) != 2 || c.GetRecommendations(chatID, "other", 10) != nil {
		t.Fatalf("recommendations were not isolated: %+v", got)
	}
	c.ClearChat(chatID)
	if c.GetRecommendations(chatID, "seed", 10) != nil {
		t.Fatal("recommendations remained after stopping playback")
	}
}
