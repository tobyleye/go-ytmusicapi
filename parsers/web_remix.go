package parsers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var trackTitleWithFeatRegex = regexp.MustCompile(`(.*?) \(feat. (.*?)\)`)

// WebRemixParser implements parsing logic for the WEB_REMIX YouTube Music client
type WebRemixParser struct{}

// NewWebRemixParser creates a new parser for WEB_REMIX client responses
func NewWebRemixParser() Parser {
	return &WebRemixParser{}
}

// ParseTrack extracts track information from a musicTwoColumnItemRenderer JSON structure
func (p *WebRemixParser) ParseTrack(trackJson interface{}) Track {
	itemRenderer := ReadValue(trackJson, []interface{}{"musicTwoColumnItemRenderer"})

	title := ReadValueString(itemRenderer, []interface{}{"title", "runs", 0, "text"})
	artists := p.parseArtists(
		ReadValue(itemRenderer, []interface{}{"subtitle", "runs"}),
	)

	titleWithFeaturedArtists := trackTitleWithFeatRegex.FindStringSubmatch(title)

	if len(titleWithFeaturedArtists) > 2 {
		title = titleWithFeaturedArtists[1]
		artist := titleWithFeaturedArtists[2]
		artists = append(artists,
			strings.Split(artist,
				fmt.Sprintf(" %s ", AND_SEPARATOR),
			)...)
	}

	videoId := ReadValueString(itemRenderer, []interface{}{
		"subtitleBadges",
		0,
		"musicDownloadStateBadgeRenderer",
		"videoId",
	})

	return Track{
		VideoId: videoId,
		Title:   title,
		Artists: artists,
		Link:    CreateTrackLink(videoId),
	}
}

// parseArtists extracts artist names from the subtitle runs array
func (p *WebRemixParser) parseArtists(artistsRow interface{}) []string {
	var artists = []string{}
	artistsRuns, _ := artistsRow.([]interface{})
	for _, artistRun := range artistsRuns {
		text := ReadValueString(artistRun, []interface{}{"text"})
		text = strings.TrimSpace(text)
		if text == DOT_SEPARATOR {
			break
		}

		if text != "" && text != AND_SEPARATOR && text != COMMA {
			artists = append(artists, text)
		}
	}
	return artists
}

// ParseSearchResults extracts tracks from a search response
func (p *WebRemixParser) ParseSearchResults(data interface{}) []Track {
	sectionListContent := ReadValue(data, []interface{}{"contents", "tabbedSearchResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents"})

	sectionListContentArray, _ := sectionListContent.([]interface{})

	musicShelfIndex := 0
	if len(sectionListContentArray) > 1 {
		musicShelfIndex = len(sectionListContentArray) - 1
	}

	musicShelfContent := ReadValue(sectionListContentArray[musicShelfIndex], []interface{}{"musicShelfRenderer", "contents"})

	var results []Track

	if content, ok := musicShelfContent.([]interface{}); ok {
		for _, item := range content {
			parsedResult := p.ParseTrack(item)
			results = append(results, parsedResult)
		}
	}

	return results
}

// ParsePlaylistDetails extracts playlist metadata and tracks from a playlist fetch response
func (p *WebRemixParser) ParsePlaylistDetails(jsonResponse interface{}) PlaylistDetails {
	playlistHeader := ReadValue(jsonResponse, []interface{}{"header", "musicEditablePlaylistDetailHeaderRenderer",
		"header", "musicElementHeaderRenderer",
	})

	if _, ok := playlistHeader.(map[string]interface{}); !ok {
		playlistHeader = ReadValue(jsonResponse, []interface{}{"header", "musicElementHeaderRenderer"})
	}

	title := ReadValueString(playlistHeader, []interface{}{"title", "runs", 0, "text"})
	totalTracksText := ReadValueString(jsonResponse, []interface{}{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicPlaylistShelfRenderer", "subFooter", "messageRenderer", "subtext", "messageSubtextRenderer", "text", "runs", 0, "text"})

	textItems := strings.Split(totalTracksText, DOT_SEPARATOR)
	totalTracks := ""

	if len(textItems) > 0 {
		totalTracks = textItems[0]
	}

	totalTracks = strings.TrimSpace(totalTracks)
	totalTracks = strings.Replace(totalTracks, " tracks", "", 1)
	totalTracks = strings.Replace(totalTracks, " songs", "", 1)
	totalTracks = strings.ReplaceAll(totalTracks, ",", "")

	totalTracksInt, _ := strconv.Atoi(totalTracks)

	playlistTracks := p.extractPlaylistTracks(jsonResponse, true)

	return PlaylistDetails{
		Title:          title,
		Description:    "",
		TotalTracks:    totalTracksInt,
		PlaylistTracks: playlistTracks,
	}
}

func (p *WebRemixParser) extractPlaylistTracks(playlistPage interface{}, isFirstPage bool) []Track {
	playlistTracks := []Track{}

	playlistHeader := ReadValue(playlistPage, []interface{}{"header", "musicEditablePlaylistDetailHeaderRenderer",
		"header", "musicElementHeaderRenderer",
	})

	_, isUserCreatedPlaylist := playlistHeader.(map[string]interface{})

	if isFirstPage {
		playlistItemsContents := ReadValue(playlistPage, []interface{}{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicPlaylistShelfRenderer", "contents"})

		if content, ok := playlistItemsContents.([]interface{}); ok {
			if isUserCreatedPlaylist {
				content = content[1:]
			}
			for _, itemContent := range content {
				item := p.ParseTrack(itemContent)
				playlistTracks = append(playlistTracks, item)
			}
		}
	} else {
		continuationItems := ReadValue(playlistPage, []interface{}{
			"continuationContents",
			"musicPlaylistShelfContinuation",
			"contents",
		})

		if items, ok := continuationItems.([]interface{}); ok {
			for _, itemContent := range items {
				track := p.ParseTrack(itemContent)
				playlistTracks = append(playlistTracks, track)
			}
		}
	}

	return playlistTracks
}

// ParsePlaylistTracks extracts tracks and continuation token from playlist tracks response
func (p *WebRemixParser) ParsePlaylistTracks(jsonResponse interface{}, isFirstPage bool) ([]Track, string) {
	playlistTracks := p.extractPlaylistTracks(jsonResponse, isFirstPage)

	nextContinuation := ""
	var continuations interface{}

	if isFirstPage {
		continuations = ReadValue(jsonResponse, []interface{}{
			"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicPlaylistShelfRenderer", "continuations",
		})
	} else {
		continuations = ReadValue(jsonResponse, []interface{}{
			"continuationContents", "musicPlaylistShelfContinuation", "continuations",
		})
	}

	if pagination, ok := continuations.([]interface{}); ok {
		lastItemIndex := len(pagination) - 1
		nextContinuation = ReadValueString(pagination[lastItemIndex], []interface{}{"nextContinuationData", "continuation"})
	}

	return playlistTracks, nextContinuation
}

// ParseUserPlaylists extracts user playlists from the library response
func (p *WebRemixParser) ParseUserPlaylists(jsonResponse interface{}, isFirstPage bool) ([]YoutubePlaylist, string) {
	var playlistItemsContents interface{}
	var nextContinuation string

	if isFirstPage {
		itemsKey := []interface{}{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicShelfRenderer", "contents"}
		playlistItemsContents = ReadValue(jsonResponse, itemsKey)
		nextContinuation = ReadValueString(jsonResponse, []interface{}{
			"contents", "singleColumnBrowseResultsRenderer", "tabs",
			0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicShelfRenderer",
			"continuations", 0, "nextContinuationData", "continuation",
		})
	} else {
		itemsKey := []interface{}{"continuationContents", "sectionListContinuation", "contents", 0, "musicShelfRenderer", "contents"}
		playlistItemsContents = ReadValue(jsonResponse, itemsKey)
		nextContinuation = ReadValueString(jsonResponse, []interface{}{
			"continuationContents", "sectionListContinuation", "contents", 0, "musicShelfRenderer", "continuations", 0, "nextContinuationData", "continuation",
		})
	}

	youtubePlaylists := []YoutubePlaylist{}

	if items, ok := playlistItemsContents.([]interface{}); ok {
		if isFirstPage && len(items) > 1 {
			items = items[1:]
		}

		for _, item := range items {
			playlist := parsePlaylistItem(item)
			youtubePlaylists = append(youtubePlaylists, playlist)
		}
	}

	return youtubePlaylists, nextContinuation
}

func parsePlaylistItem(item interface{}) YoutubePlaylist {
	itemRow := ReadValue(item, []interface{}{"musicTwoColumnItemRenderer"})
	title := ReadValueString(itemRow, []interface{}{"title", "runs", 0, "text"})

	thumbnails, _ := ReadValue(itemRow, []interface{}{"thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails"}).([]interface{})
	thumbnailUrls := []string{}

	subtitleRuns, _ := ReadValue(itemRow, []interface{}{"subtitle", "runs"}).([]interface{})

	for _, thumbnail := range thumbnails {
		thumnailMap, _ := thumbnail.(map[string]interface{})
		url, _ := thumnailMap["url"].(string)
		if url != "" {
			thumbnailUrls = append(thumbnailUrls, url)
		}
	}

	totalTracks := getPlaylistTotalTracks(subtitleRuns)

	playlistId := ReadValueString(itemRow, []interface{}{"navigationEndpoint", "browseEndpoint", "browseId"})

	if len(playlistId) > 2 && strings.HasPrefix(playlistId, "VL") {
		playlistId = playlistId[2:]
	}

	return YoutubePlaylist{
		Title:       title,
		Thumbnails:  thumbnailUrls,
		TotalTracks: totalTracks,
		PlaylistId:  playlistId,
		Url:         CreatePlaylistLink(playlistId),
	}
}

func getPlaylistTotalTracks(subtitleRuns []interface{}) string {
	lastTextRun := subtitleRuns[len(subtitleRuns)-1]
	totalTracksText := ReadValueString(lastTextRun, []interface{}{"text"})
	totalTracks := strings.Split(totalTracksText, " ")[0]
	return totalTracks
}
