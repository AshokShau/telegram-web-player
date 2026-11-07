package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"testing"
)

func TestCollectionPlaybackFollowsManualQueue(t *testing.T) {
	const chatID int64 = -9171634
	t.Cleanup(func() { cache.ChatCache.ClearChat(chatID) })
	current := &utils.PlayerCache{TrackID: "current"}
	manual := &utils.PlayerCache{TrackID: "manual"}
	first := &utils.PlayerCache{TrackID: "first"}
	second := &utils.PlayerCache{TrackID: "second"}
	cache.ChatCache.AddSong(chatID, current)
	cache.ChatCache.AddSong(chatID, manual)
	room := &RoomState{collectionNext: []*utils.PlayerCache{first, second}}

	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrCollection(chatID); got == nil || got.TrackID != manual.TrackID || len(room.collectionNext) != 2 {
		t.Fatalf("manual queue did not take priority: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrCollection(chatID); got == nil || got.TrackID != first.TrackID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("first collection track did not become current: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrCollection(chatID); got == nil || got.TrackID != second.TrackID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("second collection track did not become current: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrCollection(chatID); got != nil {
		t.Fatalf("collection continued past its end: %+v", got)
	}
}

func TestPlayingTrackCanStartCollectionContinuation(t *testing.T) {
	const chatID int64 = -9171635
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	current := &utils.PlayerCache{TrackID: "abcdefghijk"}
	cache.ChatCache.AddSong(chatID, current)
	client := voiceTestClient(chatID, 42, false)
	client.CanControl, client.CanPlay, client.AllowsWriteToPM, client.sessionActive = true, true, true, true
	tracks := []*TrackData{
		{ID: current.TrackID, URL: "https://www.youtube.com/watch?v=" + current.TrackID, Title: "First", Platform: utils.YouTube, Duration: 180},
		{ID: "lmnopqrstuv", URL: "https://www.youtube.com/watch?v=lmnopqrstuv", Title: "Second", Platform: utils.YouTube, Duration: 180},
	}
	for len(tracks) <= maxCollectionTracks {
		id := fmt.Sprintf("%011d", len(tracks))
		tracks = append(tracks, &TrackData{ID: id, URL: "https://www.youtube.com/watch?v=" + id, Title: "Later", Platform: utils.YouTube, Duration: 180})
	}
	tracks[1] = nil
	handleClientMessage(nil, client, ClientMessage{Type: "play_collection", Tracks: tracks})
	room := Manager.getOrCreate(nil, chatID)
	room.mu.Lock()
	defer room.mu.Unlock()
	if len(room.collectionNext) != maxCollectionTracks-1 || room.collectionNext[0].TrackID != tracks[2].ID || room.collectionNext[len(room.collectionNext)-1].TrackID != tracks[maxCollectionTracks].ID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("collection continuation changed the visible queue: %+v", room.collectionNext)
	}
}
