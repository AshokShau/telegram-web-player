package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"ashokshau/tg-web/internal/utils"
)

// MusicSeed is a recently played song used to find collections on YouTube Music.
type MusicSeed struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Thumbnail string `json:"thumbnail"`
}

type MusicCollection struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Thumbnail string `json:"thumbnail"`
	Kind      string `json:"kind"`
}

type MusicShelves struct {
	Albums        []MusicCollection `json:"albums"`
	Playlists     []MusicCollection `json:"playlists"`
	PlaylistTitle string            `json:"playlistTitle,omitempty"`
	Mixes         []MusicCollection `json:"mixes"`
}

var musicID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var musicBrowseID = regexp.MustCompile(`^(?:MPREb[A-Za-z0-9_-]{1,100}|VLRDCLAK[A-Za-z0-9_-]{1,100}|RDAMVM[A-Za-z0-9_-]{11})$`)

func musicPost(ctx context.Context, path string, fields map[string]any) (map[string]any, error) {
	profile := ytProfiles[2]
	payload := ytContextWithProfile(profile)
	for key, value := range fields {
		payload[key] = value
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, profile.Origin+"/youtubei/v1/"+path+"?prettyPrint=false", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	setYTHeadersWithProfile(req, profile)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("YouTube Music is unavailable")
	}
	var data map[string]any
	err = json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&data)
	return data, err
}

func musicText(node any) string { return strings.TrimSpace(youtubeMessageText(node)) }

func musicSearchItems(data map[string]any) []map[string]any {
	sections := digArray(data, "contents", "tabbedSearchResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents")
	var items []map[string]any
	for _, section := range sections {
		items = append(items, digArray(section, "itemSectionRenderer", "contents")...)
	}
	return items
}

func musicItem(item map[string]any) (MusicCollection, string) {
	renderer, _ := item["musicResponsiveListItemRenderer"].(map[string]any)
	if renderer == nil {
		return MusicCollection{}, ""
	}
	endpoint := dig(renderer, "navigationEndpoint", "browseEndpoint")
	id := digStr(endpoint, "browseId")
	page := digStr(endpoint, "browseEndpointContextSupportedConfigs", "browseEndpointContextMusicConfig", "pageType")
	title := musicText(dig(renderer, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text"))
	meta := musicText(dig(renderer, "flexColumns", 1, "musicResponsiveListItemFlexColumnRenderer", "text"))
	parts := strings.Split(meta, " • ")
	artist := ""
	if len(parts) > 1 {
		artist = parts[1]
	}
	cover := lastThumbURL(digArray(renderer, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))
	kind := ""
	switch {
	case page == "MUSIC_PAGE_TYPE_ALBUM" && strings.HasPrefix(id, "MPREb"):
		kind = "album"
	case page == "MUSIC_PAGE_TYPE_PLAYLIST" && strings.HasPrefix(id, "VLRDCLAK"):
		kind = "playlist"
	}
	return MusicCollection{ID: id, Title: title, Artist: artist, Thumbnail: cover, Kind: kind}, digStr(renderer, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
}

func musicHomePlaylists(ctx context.Context, region string) (string, []MusicCollection, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	clientContext := ytContextWithProfile(ytProfiles[2])["context"]
	if len(region) == 2 {
		clientContext.(map[string]any)["client"].(map[string]any)["gl"] = region
	}
	data, err := musicPost(requestCtx, "browse", map[string]any{"browseId": "FEmusic_home", "context": clientContext})
	if err != nil {
		return "", nil, err
	}
	sections := digArray(data, "contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents")
	for _, section := range sections {
		shelf := dig(section, "musicCarouselShelfRenderer")
		var playlists []MusicCollection
		for _, item := range digArray(shelf, "contents") {
			renderer := dig(item, "musicTwoRowItemRenderer")
			id := digStr(renderer, "navigationEndpoint", "browseEndpoint", "browseId")
			page := digStr(renderer, "navigationEndpoint", "browseEndpoint", "browseEndpointContextSupportedConfigs", "browseEndpointContextMusicConfig", "pageType")
			collection := MusicCollection{ID: id, Title: musicText(dig(renderer, "title")), Artist: musicText(dig(renderer, "subtitle")), Thumbnail: lastThumbURL(digArray(renderer, "thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails")), Kind: "playlist"}
			if page == "MUSIC_PAGE_TYPE_PLAYLIST" && musicBrowseID.MatchString(id) && collection.Title != "" && collection.Thumbnail != "" {
				playlists = append(playlists, collection)
			}
			if len(playlists) == 6 {
				break
			}
		}
		if len(playlists) >= 3 {
			return musicText(dig(shelf, "header", "musicCarouselShelfBasicHeaderRenderer", "title")), playlists, nil
		}
	}
	return "", nil, nil
}

func MusicDiscovery(ctx context.Context, seeds []MusicSeed, region ...string) (MusicShelves, error) {
	out := MusicShelves{Albums: []MusicCollection{}, Playlists: []MusicCollection{}, Mixes: []MusicCollection{}}
	seen := make(map[string]bool)
	results := make([]map[string]any, len(seeds))
	errors := make([]error, len(seeds))
	albumCandidates := make([][]MusicCollection, len(seeds))
	homeRegion := ""
	if len(region) > 0 {
		homeRegion = region[0]
	}
	var homeTitle string
	var homePlaylists []MusicCollection
	var homeErr error
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		homeTitle, homePlaylists, homeErr = musicHomePlaylists(ctx, homeRegion)
	}()
	for i, seed := range seeds {
		group.Add(1)
		go func() {
			defer group.Done()
			query := strings.TrimSpace(seed.Artist + " " + seed.Title)
			if query == "" {
				return
			}
			searchCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			results[i], errors[i] = musicPost(searchCtx, "search", map[string]any{"query": query})
		}()
	}
	group.Wait()
	var lastErr error
	for i, seed := range seeds {
		if errors[i] != nil {
			lastErr = errors[i]
			continue
		}
		mixID, mixTitle, mixCover := seed.ID, seed.Title, seed.Thumbnail
		for _, item := range musicSearchItems(results[i]) {
			collection, songID := musicItem(item)
			if musicID.MatchString(songID) && (songID == seed.ID || !musicID.MatchString(mixID)) {
				mixID = songID
				if collection.Title != "" {
					mixTitle = collection.Title
				}
				if collection.Thumbnail != "" {
					mixCover = collection.Thumbnail
				}
			}
			if collection.Kind == "" || collection.Title == "" || collection.Thumbnail == "" || seen[collection.ID] {
				continue
			}
			if collection.Kind == "album" && len(albumCandidates[i]) < 2 && strings.EqualFold(collection.Artist, seed.Artist) {
				seen[collection.ID] = true
				albumCandidates[i] = append(albumCandidates[i], collection)
			} else if collection.Kind == "playlist" && len(out.Playlists) < 6 {
				seen[collection.ID] = true
				out.Playlists = append(out.Playlists, collection)
			}
		}
		if musicID.MatchString(mixID) && !seen["RDAMVM"+mixID] && len(out.Mixes) < 4 {
			seen["RDAMVM"+mixID] = true
			out.Mixes = append(out.Mixes, MusicCollection{ID: "RDAMVM" + mixID, Title: mixTitle + " mix", Artist: seed.Artist, Thumbnail: mixCover, Kind: "mix"})
		}
	}
	albums := make([]MusicCollection, len(seeds))
	for i, candidates := range albumCandidates {
		if len(candidates) == 0 {
			continue
		}
		group.Add(1)
		go func() {
			defer group.Done()
			validationCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			for _, album := range candidates {
				tracks, err := MusicCollectionTracks(validationCtx, album)
				if err != nil {
					continue
				}
				unique := make(map[string]bool)
				for _, track := range tracks {
					if track.Duration > 0 && track.Title != "" {
						unique[track.Id] = true
					}
				}
				if len(unique) > 1 {
					albums[i] = album
					return
				}
			}
		}()
	}
	group.Wait()
	for _, album := range albums {
		if album.ID != "" {
			out.Albums = append(out.Albums, album)
		}
	}
	if len(homePlaylists) > 0 {
		out.Playlists, out.PlaylistTitle = homePlaylists, homeTitle
	}
	if len(out.Albums)+len(out.Playlists)+len(out.Mixes) == 0 && lastErr != nil {
		return out, lastErr
	}
	if len(out.Albums)+len(out.Playlists)+len(out.Mixes) == 0 && homeErr != nil {
		return out, homeErr
	}
	return out, nil
}

func MusicCollectionTracks(ctx context.Context, collection MusicCollection) ([]utils.GetUrlTrack, error) {
	if !musicBrowseID.MatchString(collection.ID) {
		return nil, errors.New("invalid YouTube Music collection")
	}
	path, fields := "browse", map[string]any{"browseId": collection.ID}
	if strings.HasPrefix(collection.ID, "RDAMVM") {
		path, fields = "next", map[string]any{"playlistId": collection.ID}
	}
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	data, err := musicPost(requestCtx, path, fields)
	if err != nil {
		return nil, err
	}
	if path == "next" {
		var tracks []utils.GetUrlTrack
		var videos []map[string]any
		var findVideos func(any)
		findVideos = func(node any) {
			switch value := node.(type) {
			case []any:
				for _, child := range value {
					findVideos(child)
				}
			case map[string]any:
				if video, ok := value["playlistPanelVideoRenderer"].(map[string]any); ok {
					videos = append(videos, video)
					return
				}
				for _, child := range value {
					findVideos(child)
				}
			}
		}
		findVideos(data["contents"])
		for _, video := range videos {
			id := digStr(video, "videoId")
			if !musicID.MatchString(id) {
				continue
			}
			tracks = append(tracks, utils.GetUrlTrack{Id: id, Url: ytWatchURL + id, Title: musicText(video["title"]), Channel: musicText(video["longBylineText"]), Thumbnail: pickYTThumb(video), Duration: parseDuration(musicText(video["lengthText"])), Platform: utils.YouTube})
		}
		return tracks, nil
	}
	var tracks []utils.GetUrlTrack
	var visit func(any)
	visit = func(node any) {
		switch value := node.(type) {
		case []any:
			for _, child := range value {
				visit(child)
			}
		case map[string]any:
			for _, shelf := range []string{"musicShelfRenderer", "musicPlaylistShelfRenderer"} {
				if content, ok := value[shelf].(map[string]any); ok {
					for _, item := range digArray(content, "contents") {
						row := dig(item, "musicResponsiveListItemRenderer")
						id := digStr(row, "playlistItemData", "videoId")
						if !musicID.MatchString(id) {
							id = digStr(row, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text", "runs", 0, "navigationEndpoint", "watchEndpoint", "videoId")
						}
						if !musicID.MatchString(id) {
							continue
						}
						tracks = append(tracks, utils.GetUrlTrack{Id: id, Url: ytWatchURL + id, Title: musicText(dig(row, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text")), Channel: musicText(dig(row, "flexColumns", 1, "musicResponsiveListItemFlexColumnRenderer", "text")), Thumbnail: lastThumbURL(digArray(row, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")), Duration: parseDuration(musicText(dig(row, "fixedColumns", 0, "musicResponsiveListItemFixedColumnRenderer", "text"))), Platform: utils.YouTube})
					}
					return
				}
			}
			for _, child := range value {
				visit(child)
			}
		}
	}
	visit(data["contents"])
	for i := range tracks {
		if tracks[i].Thumbnail == "" {
			tracks[i].Thumbnail = collection.Thumbnail
		}
	}
	return tracks, nil
}
