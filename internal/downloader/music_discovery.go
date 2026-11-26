package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
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

type MusicArtist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Thumbnail   string `json:"thumbnail"`
	Subscribers string `json:"subscribers,omitempty"`
}

type MusicArtistPage struct {
	MusicArtist
	Description string              `json:"description,omitempty"`
	Songs       []utils.GetUrlTrack `json:"songs"`
	Albums      []MusicCollection   `json:"albums"`
	Singles     []MusicCollection   `json:"singles"`
}

type MusicHomeShelf struct {
	Title    string            `json:"title"`
	Subtitle string            `json:"subtitle,omitempty"`
	Items    []MusicCollection `json:"items"`
}

type MusicShelves struct {
	Home          []MusicHomeShelf    `json:"home"`
	Trending      []utils.GetUrlTrack `json:"trending"`
	TrendingTitle string              `json:"trendingTitle,omitempty"`
	Charts        []MusicCollection   `json:"charts"`
	Languages     []MusicCollection   `json:"languages"`
	Artists       []MusicArtist       `json:"artists"`
	Albums        []MusicCollection   `json:"albums"`
	Playlists     []MusicCollection   `json:"playlists"`
	PlaylistTitle string              `json:"playlistTitle,omitempty"`
	Mixes         []MusicCollection   `json:"mixes"`
}

var musicID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var musicBrowseID = regexp.MustCompile(`^(?:MPREb[A-Za-z0-9_-]{1,100}|VL[A-Za-z0-9_-]{10,100}|RDAMVM[A-Za-z0-9_-]{11})$`)
var musicArtistID = regexp.MustCompile(`^UC[A-Za-z0-9_-]{20,24}$`)
var musicArtworkHost = regexp.MustCompile(`^lh[0-9]+\.googleusercontent\.com$`)

const musicSongsSearchParams = "EgWKAQIIAWoKEAoQAxAEEAkQBQ=="

func musicPost(ctx context.Context, path string, fields map[string]any) (map[string]any, error) {
	profile := ytProfiles[2]
	payload := ytContextWithProfile(profile)
	maps.Copy(payload, fields)
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
		items = append(items, digArray(section, "musicShelfRenderer", "contents")...)
	}
	return items
}

func searchYouTubeMusic(parent context.Context, query string, limit int) ([]utils.GetUrlTrack, error) {
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	data, err := musicPost(ctx, "search", map[string]any{
		"query":  query,
		"params": musicSongsSearchParams,
	})
	if err != nil {
		return nil, err
	}
	tracks := make([]utils.GetUrlTrack, 0, limit)
	seen := make(map[string]bool)
	for _, item := range musicSearchItems(data) {
		track := musicTrack(item)
		if track.Id == "" || track.Title == "" || track.Duration <= 0 || seen[track.Id] {
			continue
		}
		seen[track.Id] = true
		tracks = append(tracks, track)
		if len(tracks) >= limit {
			break
		}
	}
	return tracks, nil
}

func musicArtworkURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !(strings.EqualFold(u.Hostname(), "yt3.googleusercontent.com") || musicArtworkHost.MatchString(strings.ToLower(u.Hostname()))) {
		return raw
	}
	if marker := strings.LastIndex(u.Path, "="); marker >= 0 {
		u.Path = u.Path[:marker]
	}
	u.Path += "=w544-h544-l90-rj"
	return u.String()
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
	cover := musicArtworkURL(lastThumbURL(digArray(renderer, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")))
	kind := ""
	switch {
	case page == "MUSIC_PAGE_TYPE_ALBUM" && strings.HasPrefix(id, "MPREb"):
		kind = "album"
	case page == "MUSIC_PAGE_TYPE_PLAYLIST" && strings.HasPrefix(id, "VLRDCLAK"):
		kind = "playlist"
	}
	return MusicCollection{ID: id, Title: title, Artist: artist, Thumbnail: cover, Kind: kind}, digStr(renderer, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
}

func musicHomeShelves(ctx context.Context, region string) ([]MusicHomeShelf, []utils.GetUrlTrack, string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	clientContext := ytContextWithProfile(ytProfiles[2])["context"]
	if len(region) == 2 {
		clientContext.(map[string]any)["client"].(map[string]any)["gl"] = region
	}
	data, err := musicPost(requestCtx, "browse", map[string]any{"browseId": "FEmusic_home", "context": clientContext})
	if err != nil {
		return nil, nil, "", err
	}
	shelves, trending, title := parseMusicHome(data)
	return shelves, trending, title, nil
}

func parseMusicHome(data map[string]any) ([]MusicHomeShelf, []utils.GetUrlTrack, string) {
	sections := digArray(data, "contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents")
	var shelves []MusicHomeShelf
	var trending []utils.GetUrlTrack
	var trendingTitle string
	seen := make(map[string]bool)
	for _, section := range sections {
		shelf := dig(section, "musicCarouselShelfRenderer")
		header := dig(shelf, "header", "musicCarouselShelfBasicHeaderRenderer")
		title := musicText(dig(header, "title"))
		items := digArray(shelf, "contents")
		if title == "" {
			title = musicText(dig(section, "musicShelfRenderer", "title"))
			items = digArray(section, "musicShelfRenderer", "contents")
		}
		if title == "" {
			continue
		}
		var collections []MusicCollection
		localSeen := make(map[string]bool)
		for _, item := range items {
			renderer := dig(item, "musicTwoRowItemRenderer")
			id := digStr(renderer, "navigationEndpoint", "browseEndpoint", "browseId")
			if id == "" {
				id = digStr(renderer, "title", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId")
			}
			kind := ""
			switch {
			case strings.HasPrefix(id, "VL"):
				kind = "playlist"
			case strings.HasPrefix(id, "MPREb"):
				kind = "album"
			}
			collection := MusicCollection{ID: id, Title: musicText(dig(renderer, "title")), Artist: musicText(dig(renderer, "subtitle")), Thumbnail: musicArtworkURL(lastThumbURL(digArray(renderer, "thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails"))), Kind: kind}
			if kind != "" && musicBrowseID.MatchString(id) && collection.Title != "" && collection.Thumbnail != "" && !seen[id] && !localSeen[id] {
				localSeen[id] = true
				collections = append(collections, collection)
			}
			if len(collections) == 6 {
				break
			}
		}
		if len(collections) >= 3 && len(shelves) < 4 {
			shelves = append(shelves, MusicHomeShelf{Title: title, Subtitle: musicText(dig(header, "strapline")), Items: collections})
			for _, collection := range collections {
				seen[collection.ID] = true
			}
		}
		if trending == nil && strings.Contains(strings.ToLower(title), "trending") {
			for _, item := range items {
				if track := musicTrack(item); track.Id != "" && track.Title != "" {
					trending = append(trending, track)
				}
				if len(trending) == 9 {
					break
				}
			}
			if len(trending) >= 3 {
				trendingTitle = title
			} else {
				trending = nil
			}
		}
	}
	return shelves, trending, trendingTitle
}

func musicCharts(ctx context.Context, region string) (charts, languages []MusicCollection, artists []MusicArtist) {
	requestCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	clientContext := ytContextWithProfile(ytProfiles[2])["context"].(map[string]any)
	clientContext["client"].(map[string]any)["gl"] = region
	data, err := musicPost(requestCtx, "browse", map[string]any{"browseId": "FEmusic_charts", "formData": map[string]any{"selectedValues": []string{region}}, "context": clientContext})
	if err != nil {
		return
	}
	return parseMusicCharts(data, region)
}

func chartCollection(item map[string]any) MusicCollection {
	renderer := dig(item, "musicTwoRowItemRenderer")
	id := digStr(renderer, "navigationEndpoint", "browseEndpoint", "browseId")
	if id == "" {
		id = digStr(renderer, "title", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId")
	}
	if !strings.HasPrefix(id, "VL") || !musicBrowseID.MatchString(id) {
		return MusicCollection{}
	}
	return MusicCollection{ID: id, Title: musicText(dig(renderer, "title")), Thumbnail: musicArtworkURL(lastThumbURL(digArray(renderer, "thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails"))), Kind: "playlist"}
}

func chartArtist(item map[string]any) MusicArtist {
	renderer := dig(item, "musicResponsiveListItemRenderer")
	id := digStr(renderer, "navigationEndpoint", "browseEndpoint", "browseId")
	if !musicArtistID.MatchString(id) {
		id = digStr(renderer, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId")
	}
	if !musicArtistID.MatchString(id) {
		return MusicArtist{}
	}
	return MusicArtist{ID: id, Name: musicText(dig(renderer, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text")), Thumbnail: musicArtworkURL(lastThumbURL(digArray(renderer, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))), Subscribers: musicText(dig(renderer, "flexColumns", 1, "musicResponsiveListItemFlexColumnRenderer", "text"))}
}

func parseMusicCharts(data map[string]any, region string) (charts, languages []MusicCollection, artists []MusicArtist) {
	sections := digArray(data, "contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents")
	for _, section := range sections {
		shelf := dig(section, "musicCarouselShelfRenderer")
		items := digArray(shelf, "contents")
		if len(items) == 0 {
			continue
		}
		var playlistItems []MusicCollection
		for _, item := range items {
			if collection := chartCollection(item); collection.Title != "" && collection.Thumbnail != "" {
				playlistItems = append(playlistItems, collection)
			}
			if artist := chartArtist(item); artist.Name != "" && artist.Thumbnail != "" {
				artists = append(artists, artist)
			}
		}
		if len(playlistItems) > 0 {
			title := strings.ToLower(musicText(dig(shelf, "header", "musicCarouselShelfBasicHeaderRenderer", "title")))
			if region == "IN" && (strings.Contains(title, "language") || title == "" && len(charts) > 0) {
				languages = append(languages, playlistItems...)
			} else {
				charts = append(charts, playlistItems...)
			}
		}
	}
	return
}

func MusicDiscovery(ctx context.Context, seeds []MusicSeed, region ...string) (MusicShelves, error) {
	out := MusicShelves{Albums: []MusicCollection{}, Playlists: []MusicCollection{}, Mixes: []MusicCollection{}}
	seen := make(map[string]bool)
	results := make([]map[string]any, len(seeds))
	searchErrors := make([]error, len(seeds))
	albumCandidates := make([][]MusicCollection, len(seeds))
	homeRegion := "ZZ"
	if len(region) > 0 {
		if region[0] != "" {
			homeRegion = region[0]
		}
	}
	var homeErr error
	var group sync.WaitGroup
	group.Go(func() {
		out.Home, out.Trending, out.TrendingTitle, homeErr = musicHomeShelves(ctx, homeRegion)
	})
	group.Go(func() { out.Charts, out.Languages, out.Artists = musicCharts(ctx, homeRegion) })
	for i, seed := range seeds {
		group.Go(func() {
			query := strings.TrimSpace(seed.Artist + " " + seed.Title)
			if query == "" {
				return
			}
			searchCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			results[i], searchErrors[i] = musicPost(searchCtx, "search", map[string]any{"query": query})
		})
	}
	group.Wait()
	var lastErr error
	for i, seed := range seeds {
		if searchErrors[i] != nil {
			lastErr = searchErrors[i]
			continue
		}
		mixID, mixTitle, mixCover := seed.ID, seed.Title, musicArtworkURL(seed.Thumbnail)
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
		group.Go(func() {
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
		})
	}
	group.Wait()
	for _, album := range albums {
		if album.ID != "" {
			out.Albums = append(out.Albums, album)
		}
	}
	if len(out.Home)+len(out.Trending)+len(out.Albums)+len(out.Playlists)+len(out.Mixes)+len(out.Charts)+len(out.Languages)+len(out.Artists) == 0 {
		if lastErr != nil {
			return out, lastErr
		}
		if homeErr != nil {
			return out, homeErr
		}
		return out, errors.New("no music collections available")
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
			tracks = append(tracks, utils.GetUrlTrack{Id: id, Url: ytWatchURL + id, Title: musicText(video["title"]), Channel: musicText(video["longBylineText"]), Thumbnail: musicArtworkURL(bestYouTubeThumbnail(id, digArray(video, "thumbnail", "thumbnails"))), Duration: parseDuration(musicText(video["lengthText"])), Platform: utils.YouTube})
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
						if track := musicTrack(item); track.Id != "" {
							tracks = append(tracks, track)
						}
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
	return tracks, nil
}

func musicTrack(item map[string]any) utils.GetUrlTrack {
	row := dig(item, "musicResponsiveListItemRenderer")
	id := digStr(row, "playlistItemData", "videoId")
	if !musicID.MatchString(id) {
		id = digStr(row, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
	}
	if !musicID.MatchString(id) {
		id = digStr(row, "navigationEndpoint", "watchEndpoint", "videoId")
	}
	if !musicID.MatchString(id) {
		id = digStr(row, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text", "runs", 0, "navigationEndpoint", "watchEndpoint", "videoId")
	}
	if !musicID.MatchString(id) {
		return utils.GetUrlTrack{}
	}
	meta := dig(row, "flexColumns", 1, "musicResponsiveListItemFlexColumnRenderer", "text")
	artist := musicText(meta)
	metaRuns := digArray(meta, "runs")
	for _, run := range metaRuns {
		name := strings.TrimSpace(safeString(run["text"]))
		if name != "" && name != "•" && !strings.EqualFold(name, "Song") {
			artist = name
			break
		}
	}
	durationText := musicText(dig(row, "fixedColumns", 0, "musicResponsiveListItemFixedColumnRenderer", "text"))
	if durationText == "" && len(metaRuns) > 0 {
		durationText = strings.TrimSpace(safeString(metaRuns[len(metaRuns)-1]["text"]))
	}
	var duration int32
	if strings.Contains(durationText, ":") {
		duration = parseDuration(durationText)
	}
	return utils.GetUrlTrack{Id: id, Url: ytWatchURL + id, Title: musicText(dig(row, "flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text")), Channel: artist, Thumbnail: musicArtworkURL(bestYouTubeThumbnail(id, digArray(row, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))), Duration: duration, Platform: utils.YouTube}
}

func MusicArtistDetails(ctx context.Context, id string) (MusicArtistPage, error) {
	if !musicArtistID.MatchString(id) {
		return MusicArtistPage{}, errors.New("invalid YouTube Music artist")
	}
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	data, err := musicPost(requestCtx, "browse", map[string]any{"browseId": id})
	if err != nil {
		return MusicArtistPage{}, err
	}
	page := MusicArtistPage{ID: id, Songs: []utils.GetUrlTrack{}, Albums: []MusicCollection{}, Singles: []MusicCollection{}}
	header := dig(data, "header", "musicImmersiveHeaderRenderer")
	if header == nil {
		header = dig(data, "header", "musicVisualHeaderRenderer")
	}
	page.Name = musicText(dig(header, "title"))
	page.Description = musicText(dig(header, "description"))
	page.Subscribers = musicText(dig(header, "subscriptionButton", "subscribeButtonRenderer", "subscriberCountText"))
	page.Thumbnail = musicArtworkURL(lastThumbURL(digArray(header, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")))
	if page.Thumbnail == "" {
		page.Thumbnail = musicArtworkURL(lastThumbURL(digArray(header, "foregroundThumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")))
	}
	sections := digArray(data, "contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents")
	for _, section := range sections {
		if shelf := dig(section, "musicShelfRenderer"); shelf != nil && strings.Contains(strings.ToLower(musicText(dig(shelf, "title"))), "song") {
			for _, item := range digArray(shelf, "contents") {
				if track := musicTrack(item); track.Id != "" {
					page.Songs = append(page.Songs, track)
				}
			}
		}
		shelf := dig(section, "musicCarouselShelfRenderer")
		title := strings.ToLower(musicText(dig(shelf, "header", "musicCarouselShelfBasicHeaderRenderer", "title")))
		if !strings.Contains(title, "album") && !strings.Contains(title, "single") {
			continue
		}
		for _, item := range digArray(shelf, "contents") {
			renderer := dig(item, "musicTwoRowItemRenderer")
			id := digStr(renderer, "navigationEndpoint", "browseEndpoint", "browseId")
			if !strings.HasPrefix(id, "MPREb") || !musicBrowseID.MatchString(id) {
				continue
			}
			collection := MusicCollection{ID: id, Title: musicText(dig(renderer, "title")), Artist: page.Name, Thumbnail: musicArtworkURL(lastThumbURL(digArray(renderer, "thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails"))), Kind: "album"}
			if collection.Title == "" {
				continue
			}
			if strings.Contains(title, "single") {
				page.Singles = append(page.Singles, collection)
			} else {
				page.Albums = append(page.Albums, collection)
			}
		}
	}
	if page.Name == "" {
		return MusicArtistPage{}, errors.New("YouTube Music artist is unavailable")
	}
	return page, nil
}
