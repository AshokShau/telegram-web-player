package webapp

import (
	"ashokshau/tg-web/internal/cache"
	"ashokshau/tg-web/internal/utils"
	"context"
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
	room := &RoomState{nextTracks: []*utils.PlayerCache{first, second}}

	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != manual.TrackID || len(room.nextTracks) != 2 {
		t.Fatalf("manual queue did not take priority: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != first.TrackID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("first collection track did not become current: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != second.TrackID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("second collection track did not become current: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got != nil {
		t.Fatalf("collection continued past its end: %+v", got)
	}
}

func TestEnablingAutoplayReplacesCollectionContinuation(t *testing.T) {
	const chatID int64 = -9171638
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "current"})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "manual"})
	tracks := make([]*utils.PlayerCache, maxCollectionTracks)
	for i := range tracks {
		tracks[i] = &utils.PlayerCache{TrackID: fmt.Sprintf("collection-%d", i)}
	}
	SetPlaybackSequence(nil, chatID, tracks)
	room := Manager.getOrCreate(nil, chatID)
	room.autoplayUsed = true
	SetAutoplay(nil, chatID, true)
	if !cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 0 || room.autoplayUsed || room.autoplaySeed == nil || room.autoplaySeed.TrackID != "current" || cache.ChatCache.GetQueueLength(chatID) != 2 {
		t.Fatalf("enabling autoplay did not replace pending collection: %+v", room.nextTracks)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != "manual" {
		t.Fatalf("manual queue lost priority: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got != nil {
		t.Fatalf("old collection resumed after autoplay was enabled: %+v", got)
	}
}

func TestAutoplayKeepsItsRecommendationSeedAcrossTrackChanges(t *testing.T) {
	const chatID int64 = -9171640
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	seedURL := "https://www.youtube.com/watch?v=abcdefghijk"
	current := &utils.PlayerCache{TrackID: "abcdefghijk", URL: seedURL, Name: "Mood (Lofi)", Platform: utils.YouTube}
	cache.ChatCache.AddSong(chatID, current)
	SetAutoplay(nil, chatID, true)
	current.URL = "https://www.youtube.com/watch?v=changed12345"
	cache.ChatCache.RemoveCurrentSong(chatID)
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "next", URL: "https://www.youtube.com/watch?v=lmnopqrstuv", User: utils.AutoPlay})
	room := Manager.getOrCreate(nil, chatID)
	if got := recommendationSeed(room.autoplaySeed); got != seedURL {
		t.Fatalf("autoplay recommendation seed changed with playback: %q", got)
	}
}

func TestAutoplayReusesSimilarTracksForTheSameSong(t *testing.T) {
	const chatID int64 = -9171639
	t.Cleanup(func() {
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	room := Manager.getOrCreate(nil, chatID)
	seed := "https://www.youtube.com/watch?v=abcdefghijk"
	room.recommendations = map[string]*recommendationEntry{seed: {limit: 10, tracks: []utils.GetUrlTrack{{Id: "lmnopqrstuv", Title: "Related song"}}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tracks, err := roomRecommendations(ctx, nil, chatID, seed, 10)
	if err != nil || len(tracks) != 1 || tracks[0].Id != "lmnopqrstuv" {
		t.Fatalf("cached similar tracks were not reused: tracks=%+v err=%v", tracks, err)
	}
}

func TestPreparedPlaybackOrderAndAutoplayToggle(t *testing.T) {
	const chatID int64 = -9171636
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "current"})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "manual"})
	SetAutoplay(nil, chatID, true)
	SetPlaybackSequence(nil, chatID, []*utils.PlayerCache{
		{TrackID: "collection"},
		{TrackID: "auto", User: utils.AutoPlay},
	})
	room := Manager.getOrCreate(nil, chatID)
	for _, want := range []string{"manual", "collection", "auto"} {
		cache.ChatCache.RemoveCurrentSong(chatID)
		if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != want {
			t.Fatalf("next track = %+v, want %q", got, want)
		}
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got != nil {
		t.Fatalf("prepared playback continued past its batch: %+v", got)
	}

	SetPlaybackSequence(nil, chatID, []*utils.PlayerCache{{TrackID: "collection2"}, {TrackID: "auto2", User: utils.AutoPlay}})
	ctx, cancel := context.WithCancel(context.Background())
	room.autoplayCancel = cancel
	SetAutoplay(nil, chatID, false)
	if cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 1 || ctx.Err() != context.Canceled {
		t.Fatalf("disabling autoplay removed collection tracks: %+v", room.nextTracks)
	}
	if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != "collection2" {
		t.Fatalf("collection did not continue with autoplay off: %+v", got)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got != nil {
		t.Fatalf("autoplay continued after being disabled: %+v", got)
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
	if len(room.nextTracks) != maxCollectionTracks-1 || room.nextTracks[0].TrackID != tracks[2].ID || room.nextTracks[len(room.nextTracks)-1].TrackID != tracks[maxCollectionTracks].ID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("collection continuation changed the visible queue: %+v", room.nextTracks)
	}
}
