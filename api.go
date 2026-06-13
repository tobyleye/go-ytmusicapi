package ytmusicapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/tobyleye/go-ytmusicapi/parsers"
)

// Re-export types from parsers package
type Track = parsers.Track
type YoutubePlaylist = parsers.YoutubePlaylist
type PlaylistDetails = parsers.PlaylistDetails
type PlaylistAllTracksResponse = parsers.PlaylistAllTracksResponse
type PlaylistTracksResponse = parsers.PlaylistTracksResponse
type CreatedPlaylist = parsers.CreatedPlaylist
type PlaylistPageResponse = parsers.PlaylistPageResponse

// SearchQuery specifies the parameters for a track search
type SearchQuery struct {
	Title   string
	Artists []string
	Type    string
}

const ytmDomain = "https://music.youtube.com"

var defaultHeaders = map[string]string{
	"accept":       "*/*",
	"content-type": "application/json",
	"origin":       ytmDomain,
}

type webClient struct {
	name    string
	version string
	parser  parsers.Parser
}

var html5Client = &webClient{
	name:    "TVHTML5",
	version: "7.20240925.00.00",
	parser:  parsers.NewTVHTML5Parser(),
}

var html5EmbeddedClient = &webClient{
	name:    "TVHTML5_SIMPLY_EMBEDDED_PLAYER",
	version: "2.0",
	parser:  nil,
}

var webRemixClient = &webClient{
	name:    "WEB_REMIX",
	version: "1.20250910.01.00",
	parser:  parsers.NewWebRemixParser(),
}

func getSearchParams(filter, scope string, ignoreSpelling bool) string {
	filteredParam1 := "EgWKAQ"
	var params string
	var param1, param2, param3 string

	if filter == "" && scope == "" && !ignoreSpelling {
		return params
	}

	if scope == "uploads" {
		params = "agIYAw%3D%3D"
	}

	if scope == "library" {
		if filter != "" {
			param1 = filteredParam1
			param2 = getParam2(filter)
			param3 = "AWoKEAUQCRADEAoYBA%3D%3D"
		} else {
			params = "agIYBA%3D%3D"
		}
	}

	if scope == "" && filter != "" {
		if filter == "playlists" {
			params = "Eg-KAQwIABAAGAAgACgB"
			if !ignoreSpelling {
				params += "MABqChAEEAMQCRAFEAo%3D"
			} else {
				params += "MABCAggBagoQBBADEAkQBRAK"
			}
		} else if filter == "featured_playlists" || filter == "community_playlists" {
			param1 = "EgeKAQQoA"
			if filter == "featured_playlists" {
				param2 = "Dg"
			} else {
				param2 = "EA"
			}

			if !ignoreSpelling {
				param3 = "BagwQDhAKEAMQBBAJEAU%3D"
			} else {
				param3 = "BQgIIAWoMEA4QChADEAQQCRAF"
			}
		} else {
			param1 = filteredParam1
			param2 = getParam2(filter)
			if !ignoreSpelling {
				param3 = "AWoMEA4QChADEAQQCRAF"
			} else {
				param3 = "AUICCAFqDBAOEAoQAxAEEAkQBQ%3D%3D"
			}
		}
	}

	if scope == "" && filter == "" && ignoreSpelling {
		params = "EhGKAQ4IARABGAEgASgAOAFAAUICCAE%3D"
	}

	if params != "" {
		return params
	}

	return param1 + param2 + param3
}

func getParam2(filter string) string {
	filterParams := map[string]string{
		"songs":     "II",
		"videos":    "IQ",
		"albums":    "IY",
		"artists":   "Ig",
		"playlists": "Io",
		"profiles":  "JY",
		"podcasts":  "JQ",
		"episodes":  "JI",
	}
	return filterParams[filter]
}

func sendRequest(httpClient *http.Client, client *webClient, endpoint string, body map[string]interface{}) (interface{}, error) {
	url := fmt.Sprintf("%s/youtubei/v1/%s?alt=json", ytmDomain, endpoint)

	body["context"] = map[string]interface{}{
		"client": map[string]interface{}{
			"clientName":    client.name,
			"clientVersion": client.version,
		},
		"user": map[string]interface{}{},
	}

	var jsonResponse interface{}

	builder := requests.URL(url).Client(httpClient)
	for key, val := range defaultHeaders {
		builder.Header(key, val)
	}

	err := builder.BodyJSON(&body).ToJSON(&jsonResponse).Fetch(context.Background())
	return jsonResponse, err
}

// Search returns tracks matching the given query
func Search(client *http.Client, searchQuery SearchQuery) ([]Track, error) {
	query := searchQuery.Title + " by " + strings.Join(searchQuery.Artists, ", ")
	params := getSearchParams("songs", "", true)

	body := map[string]interface{}{"query": query}
	if params != "" {
		body["params"] = params
	}

	data, err := sendRequest(client, html5Client, "search", body)
	if err != nil {
		return nil, err
	}

	return html5Client.parser.ParseSearchResults(data), nil
}

// SearchOne returns the top matching track for the given query
func SearchOne(client *http.Client, searchQuery SearchQuery) (Track, error) {
	results, err := Search(client, searchQuery)
	if err != nil {
		return Track{}, err
	}
	if len(results) == 0 {
		return Track{}, nil
	}
	return results[0], nil
}

// FetchPlaylist returns the details of a playlist by ID
func FetchPlaylist(client *http.Client, playlistId string) (PlaylistDetails, error) {
	browseId := playlistId
	if !strings.HasPrefix(browseId, "VL") {
		browseId = "VL" + browseId
	}

	body := map[string]interface{}{"browseId": browseId}
	jsonResponse, err := sendRequest(client, html5Client, "browse", body)
	if err != nil {
		return PlaylistDetails{}, err
	}

	playlistDetails := html5Client.parser.ParsePlaylistDetails(jsonResponse)
	playlistDetails.Link = parsers.CreatePlaylistLink(playlistId)
	return playlistDetails, nil
}

// FetchPlaylistTracks returns a page of tracks for a playlist.
// Pass an empty continuation string to fetch the first page.
func FetchPlaylistTracks(client *http.Client, playlistId string, continuation string) (PlaylistTracksResponse, error) {
	browseId := playlistId
	if !strings.HasPrefix(browseId, "VL") {
		browseId = "VL" + browseId
	}

	body := map[string]interface{}{}
	if continuation != "" {
		body["continuation"] = continuation
	} else {
		body["browseId"] = browseId
	}

	jsonResponse, err := sendRequest(client, html5Client, "browse", body)
	if err != nil {
		return PlaylistTracksResponse{}, err
	}

	tracks, nextContinuation := html5Client.parser.ParsePlaylistTracks(jsonResponse, continuation == "")
	return PlaylistTracksResponse{
		NextContinuation: nextContinuation,
		Tracks:           tracks,
	}, nil
}

// FetchAllPlaylistTracks fetches every track in a playlist, paginating automatically
func FetchAllPlaylistTracks(client *http.Client, playlistId string) (PlaylistAllTracksResponse, error) {
	tracks := []Track{}
	nextContinuation := ""

	for {
		page, err := FetchPlaylistTracks(client, playlistId, nextContinuation)
		if err != nil {
			return PlaylistAllTracksResponse{}, err
		}
		tracks = append(tracks, page.Tracks...)
		nextContinuation = page.NextContinuation
		if nextContinuation == "" {
			break
		}
	}

	return PlaylistAllTracksResponse{
		Total:  len(tracks),
		Tracks: tracks,
	}, nil
}

// FetchUserPlaylists returns a page of the authenticated user's playlists.
// Pass an empty continuation string to fetch the first page.
func FetchUserPlaylists(httpClient *http.Client, continuation string) (PlaylistPageResponse, error) {
	var body map[string]interface{}
	if continuation == "" {
		body = map[string]interface{}{"browseId": "FEmusic_liked_playlists"}
	} else {
		body = map[string]interface{}{"continuation": continuation}
	}

	jsonResponse, err := sendRequest(httpClient, html5Client, "browse", body)
	if err != nil {
		return PlaylistPageResponse{}, err
	}

	playlists, nextContinuation := html5Client.parser.ParseUserPlaylists(jsonResponse, continuation == "")
	return PlaylistPageResponse{
		Continuation: nextContinuation,
		Playlists:    playlists,
	}, nil
}

// CreatePlaylist creates a new private playlist and returns its details
func CreatePlaylist(client *http.Client, title string, description string, videoIds []string) (CreatedPlaylist, error) {
	body := map[string]interface{}{
		"title":         title,
		"description":   description,
		"privacyStatus": "PRIVATE",
		"videoIds":      videoIds,
	}
	data, err := sendRequest(client, html5EmbeddedClient, "playlist/create", body)
	if err != nil {
		return CreatedPlaylist{}, err
	}

	playlistId := ReadValueString(data, []interface{}{"playlistId"})
	return CreatedPlaylist{
		PlaylistId:  playlistId,
		Link:        parsers.CreatePlaylistLink(playlistId),
		Title:       title,
		Description: description,
	}, nil
}

// AddTracksToPlaylist adds the given video IDs to an existing playlist
func AddTracksToPlaylist(client *http.Client, playlistId string, videoIds []string) error {
	if len(videoIds) == 0 {
		return nil
	}

	actions := make([]map[string]string, 0, len(videoIds))
	for _, videoId := range videoIds {
		actions = append(actions, map[string]string{
			"action":       "ACTION_ADD_VIDEO",
			"addedVideoId": videoId,
		})
	}

	body := map[string]interface{}{
		"playlistId": playlistId,
		"actions":    actions,
	}

	_, err := sendRequest(client, html5EmbeddedClient, "browse/edit_playlist", body)
	return err
}

// FetchLikedPlaylist returns the authenticated user's "Liked Music" playlist
func FetchLikedPlaylist(client *http.Client) (YoutubePlaylist, error) {
	const likedPlaylistId = "LM"
	playlistDetails, err := FetchPlaylist(client, likedPlaylistId)
	if err != nil {
		return YoutubePlaylist{}, err
	}

	return YoutubePlaylist{
		Title:       playlistDetails.Title,
		Thumbnails:  playlistDetails.Thumbnails,
		TotalTracks: fmt.Sprintf("%d", playlistDetails.TotalTracks),
		PlaylistId:  likedPlaylistId,
		Url:         parsers.CreatePlaylistLink(likedPlaylistId),
	}, nil
}
