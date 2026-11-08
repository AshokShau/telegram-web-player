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

func TestSelectUpNextKeepsQueuePriorityAndCollectionRemainder(t *testing.T) {
	const chatID int64 = -9171642
	t.Cleanup(func() { cache.ChatCache.ClearChat(chatID) })
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "current"})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "queued"})
	room := &RoomState{nextTracks: []*utils.PlayerCache{{TrackID: "first"}, {TrackID: "selected"}, {TrackID: "last"}}}
	room.isTransitioning = true
	if room.selectUpNext(chatID, 2, "selected") {
		t.Fatal("selected a track during a playback transition")
	}
	room.isTransitioning = false
	if room.selectUpNext(chatID, 1, "wrong") || !room.selectUpNext(chatID, 2, "selected") {
		t.Fatal("failed to select a prepared track by its current index and ID")
	}
	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) != 3 || queue[1].TrackID != "selected" || queue[2].TrackID != "queued" || len(room.nextTracks) != 1 || room.nextTracks[0].TrackID != "last" {
		t.Fatalf("selecting Up next changed the wrong tracks: queue=%+v prepared=%+v", queue, room.nextTracks)
	}
	if !room.selectUpNext(chatID, 1, "queued") || cache.ChatCache.GetQueue(chatID)[1].TrackID != "queued" {
		t.Fatal("queued track did not move to the front")
	}
	for _, want := range []string{"queued", "selected", "last"} {
		cache.ChatCache.RemoveCurrentSong(chatID)
		if next := room.nextQueuedOrPrepared(chatID); next == nil || next.TrackID != want {
			t.Fatalf("next track = %+v, want %q", next, want)
		}
	}
}

func TestRemoveAndClearUpNext(t *testing.T) {
	const chatID int64 = -9171643
	t.Cleanup(func() { cache.ChatCache.ClearChat(chatID) })
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "current"})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "manual"})
	room := &RoomState{nextTracks: []*utils.PlayerCache{{TrackID: "first"}, {TrackID: "last"}}}
	if room.removeUpNext(chatID, 2, "wrong") || !room.removeUpNext(chatID, 2, "first") {
		t.Fatal("failed to remove the selected collection track")
	}
	if !room.removeUpNext(chatID, 1, "manual") || len(cache.ChatCache.GetQueue(chatID)) != 1 {
		t.Fatal("failed to remove the manually queued track")
	}
	room.clearUpNext(chatID)
	if len(room.nextTracks) != 0 || cache.ChatCache.GetPlayingTrack(chatID).TrackID != "current" {
		t.Fatal("clearing Up next changed the playing track")
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
	SetPlaybackSequence(nil, chatID, tracks, true)
	room := Manager.getOrCreate(nil, chatID)
	setAutoplay(nil, chatID, true)
	if !cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 0 || cache.ChatCache.GetQueueLength(chatID) != 2 {
		t.Fatalf("enabling autoplay did not replace collection continuation: %+v", room.nextTracks)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != "manual" {
		t.Fatalf("manual queue lost priority: %+v", got)
	}
	if len(room.nextTracks) != 0 {
		t.Fatalf("collection reappeared after manual queue: %+v", room.nextTracks)
	}
}

func TestSequenceReplacedAcrossAutoplayModes(t *testing.T) {
	const chatID int64 = -9171640
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	SetPlaybackSequence(nil, chatID, []*utils.PlayerCache{{TrackID: "old"}}, true)
	SetPlaybackSequence(nil, chatID, []*utils.PlayerCache{{TrackID: "new"}}, true)
	room := Manager.getOrCreate(nil, chatID)
	if len(room.nextTracks) != 1 || room.nextTracks[0].TrackID != "new" {
		t.Fatalf("autoplay off did not replace sequence: %+v", room.nextTracks)
	}
	setAutoplay(nil, chatID, true)
	room.autoplaySeed = &utils.PlayerCache{TrackID: "seed"}
	room.nextTracks = []*utils.PlayerCache{{TrackID: "auto", User: utils.AutoPlay}}
	SetPlaybackSequence(nil, chatID, nil, false)
	if !cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 0 || room.autoplaySeed != nil {
		t.Fatalf("manual play kept stale autoplay continuation: %+v", room.nextTracks)
	}
	room.nextTracks = []*utils.PlayerCache{{TrackID: "auto2", User: utils.AutoPlay}}
	SetPlaybackSequence(nil, chatID, []*utils.PlayerCache{{TrackID: "first"}, {TrackID: "second"}}, true)
	if cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 2 || room.nextTracks[0].TrackID != "first" || room.nextTracks[1].TrackID != "second" {
		t.Fatalf("collection did not replace autoplay: %+v", room.nextTracks)
	}
}

func TestAutoplayReusesSimilarTracksForTheSameSong(t *testing.T) {
	const chatID int64 = -9171639
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	seed := "https://www.youtube.com/watch?v=abcdefghijk"
	cache.ChatCache.SetRecommendations(chatID, seed, []utils.GetUrlTrack{{Id: "lmnopqrstuv", Title: "Related song"}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tracks, err := roomRecommendations(ctx, chatID, seed, 10)
	if err != nil || len(tracks) != 1 || tracks[0].Id != "lmnopqrstuv" {
		t.Fatalf("cached similar tracks were not reused: tracks=%+v err=%v", tracks, err)
	}
}

func TestManualPlayReplacesAutoplayBatch(t *testing.T) {
	const chatID int64 = -9171641
	t.Cleanup(func() {
		cache.ChatCache.ClearChat(chatID)
		Manager.mu.Lock()
		delete(Manager.rooms, chatID)
		Manager.mu.Unlock()
	})
	cache.ChatCache.AddSong(chatID, &utils.PlayerCache{TrackID: "old", URL: "https://www.youtube.com/watch?v=abcdefghijk"})
	setAutoplay(nil, chatID, true)
	room := Manager.getOrCreate(nil, chatID)
	room.nextTracks = []*utils.PlayerCache{{TrackID: "old-auto", User: utils.AutoPlay}}
	room.autoplaySeed = &utils.PlayerCache{TrackID: "old"}
	SetPlaybackSequence(nil, chatID, nil, false)
	if !cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 0 || room.autoplaySeed != nil {
		t.Fatalf("manual play did not reset autoplay batch: %+v", room.nextTracks)
	}
	cache.ChatCache.AddSongToFront(chatID, &utils.PlayerCache{TrackID: "manual", URL: "https://www.youtube.com/watch?v=lmnopqrstuv"})
	cache.ChatCache.RemoveCurrentSong(chatID)
	if next := room.nextQueuedOrPrepared(chatID); next == nil || recommendationSeed(next) != "https://www.youtube.com/watch?v=lmnopqrstuv" {
		t.Fatalf("next autoplay seed is not the manually played track: %+v", next)
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
	setAutoplay(nil, chatID, true)
	room := Manager.getOrCreate(nil, chatID)
	room.nextTracks = []*utils.PlayerCache{{TrackID: "auto", User: utils.AutoPlay}, {TrackID: "auto2", User: utils.AutoPlay}}
	for _, want := range []string{"manual"} {
		cache.ChatCache.RemoveCurrentSong(chatID)
		if got := room.nextQueuedOrPrepared(chatID); got == nil || got.TrackID != want {
			t.Fatalf("next track = %+v, want %q", got, want)
		}
	}
	if len(room.nextTracks) != 0 {
		t.Fatalf("manual queue did not clear stale autoplay batch: %+v", room.nextTracks)
	}
	cache.ChatCache.RemoveCurrentSong(chatID)
	if got := room.nextQueuedOrPrepared(chatID); got != nil {
		t.Fatalf("prepared playback continued past its batch: %+v", got)
	}

	room.nextTracks = []*utils.PlayerCache{{TrackID: "auto3", User: utils.AutoPlay}}
	ctx, cancel := context.WithCancel(context.Background())
	room.autoplayCancel = cancel
	setAutoplay(nil, chatID, false)
	if cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != 0 || ctx.Err() != context.Canceled {
		t.Fatalf("disabling autoplay kept autoplay tracks: %+v", room.nextTracks)
	}
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
	setAutoplay(nil, chatID, true)
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
	if cache.ChatCache.GetAutoplay(chatID) || len(room.nextTracks) != maxCollectionTracks-1 || room.nextTracks[0].TrackID != tracks[2].ID || room.nextTracks[len(room.nextTracks)-1].TrackID != tracks[maxCollectionTracks].ID || cache.ChatCache.GetQueueLength(chatID) != 1 {
		t.Fatalf("collection continuation changed the visible queue: %+v", room.nextTracks)
	}
}
