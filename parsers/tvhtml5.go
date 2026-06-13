package parsers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var tvhtml5TrackTitleWithFeatRegex = regexp.MustCompile(`(.*?) \(feat. (.*?)\)`)

// TVHTML5Parser implements parsing logic for the TVHTML5_SIMPLY_EMBEDDED_PLAYER YouTube Music client
type TVHTML5Parser struct{}

// NewTVHTML5Parser creates a parser for TVHTML5 client responses
func NewTVHTML5Parser() Parser {
	return &TVHTML5Parser{}
}

// ParseSearchResults extracts tracks from a search response
func (p *TVHTML5Parser) ParseSearchResults(data interface{}) []Track {
	tracks := []Track{}

	sections := ReadValue(data, []interface{}{
		"contents", "sectionListRenderer", "contents",
	})

	if sectionsList, ok := sections.([]interface{}); ok {
		for _, section := range sectionsList {
			items := ReadValue(section, []interface{}{
				"shelfRenderer", "content", "horizontalListRenderer", "items",
			})

			if itemsList, ok := items.([]interface{}); ok {
				for _, item := range itemsList {
					track := parseTrackItem(item)
					if track.VideoId != "" {
						tracks = append(tracks, track)
					}
				}
			}
		}
	}

	return tracks
}

// ParsePlaylistDetails extracts playlist metadata and tracks from a playlist fetch response
func (p *TVHTML5Parser) ParsePlaylistDetails(jsonResponse interface{}) PlaylistDetails {
	metadata := ReadValue(jsonResponse, []interface{}{
		"contents", "tvBrowseRenderer", "content", "tvSurfaceContentRenderer",
		"content", "twoColumnRenderer", "leftColumn", "entityMetadataRenderer",
	})

	title := ReadValueString(metadata, []interface{}{"title", "simpleText"})
	description := ReadValueString(metadata, []interface{}{"description", "simpleText"})

	bylineItems := ReadValue(metadata, []interface{}{"bylines", 0, "lineRenderer", "items"})

	totalTracks := 0
	if items, ok := bylineItems.([]interface{}); ok {
		for _, item := range items {
			runs := ReadValue(item, []interface{}{"lineItemRenderer", "text", "runs"})
			if runsArray, ok := runs.([]interface{}); ok && len(runsArray) > 0 {
				count := ReadValueString(runsArray[0], []interface{}{"text"})
				if count != "" {
					if len(runsArray) > 1 {
						suffix := ReadValueString(runsArray[1], []interface{}{"text"})
						if strings.Contains(suffix, "video") || strings.Contains(suffix, "song") {
							totalTracks = cleanTrackCount(count)
							break
						}
					}
				}
			}
		}
	}

	thumbnails := []string{}
	firstTrack := ReadValue(jsonResponse, []interface{}{
		"contents", "tvBrowseRenderer", "content", "tvSurfaceContentRenderer",
		"content", "twoColumnRenderer", "rightColumn", "playlistVideoListRenderer",
		"contents", 0, "tileRenderer", "header", "tileHeaderRenderer", "thumbnail", "thumbnails",
	})

	if thumbsList, ok := firstTrack.([]interface{}); ok && len(thumbsList) > 0 {
		for _, thumb := range thumbsList {
			if thumbMap, ok := thumb.(map[string]interface{}); ok {
				if url, ok := thumbMap["url"].(string); ok && url != "" {
					thumbnails = append(thumbnails, url)
				}
			}
		}
	}

	tracks, _ := p.ParsePlaylistTracks(jsonResponse, true)

	return PlaylistDetails{
		Title:          title,
		Description:    description,
		TotalTracks:    totalTracks,
		PlaylistTracks: tracks,
		Link:           "",
		Thumbnails:     thumbnails,
	}
}

// parseTrackItem extracts track information from a tileRenderer
func parseTrackItem(item interface{}) Track {
	tileRenderer := ReadValue(item, []interface{}{"tileRenderer"})
	if tileRenderer == nil {
		return Track{}
	}

	videoId := ReadValueString(tileRenderer, []interface{}{"contentId"})

	title := ReadValueString(tileRenderer, []interface{}{
		"metadata", "tileMetadataRenderer", "title", "simpleText",
	})

	artistName := ReadValueString(tileRenderer, []interface{}{
		"metadata", "tileMetadataRenderer", "lines", 0,
		"lineRenderer", "items", 0, "lineItemRenderer", "text", "runs", 0, "text",
	})

	if artistName == "" {
		artistName = ReadValueString(tileRenderer, []interface{}{
			"metadata", "tileMetadataRenderer", "lines", 0,
			"lineRenderer", "items", 0, "lineItemRenderer", "text", "simpleText",
		})
	}

	artists := []string{}
	if artistName != "" {
		artists = append(artists, artistName)
	}

	cleanTitle, featuredArtists := parseFeaturedArtists(title)
	if len(featuredArtists) > 0 {
		title = cleanTitle
		artists = append(artists, featuredArtists...)
	}

	return Track{
		VideoId: videoId,
		Title:   title,
		Artists: artists,
		Link:    CreateTrackLink(videoId),
	}
}

// ParsePlaylistTracks extracts tracks and continuation token from playlist tracks response
func (p *TVHTML5Parser) ParsePlaylistTracks(jsonResponse interface{}, isFirstPage bool) ([]Track, string) {
	var contents interface{}
	var nextContinuation string

	if isFirstPage {
		contents = ReadValue(jsonResponse, []interface{}{
			"contents", "tvBrowseRenderer", "content", "tvSurfaceContentRenderer",
			"content", "twoColumnRenderer", "rightColumn", "playlistVideoListRenderer",
			"contents",
		})

		nextContinuation = ReadValueString(jsonResponse, []interface{}{
			"contents", "tvBrowseRenderer", "content", "tvSurfaceContentRenderer",
			"content", "twoColumnRenderer", "rightColumn", "playlistVideoListRenderer",
			"continuations", 0, "nextContinuationData", "continuation",
		})
	} else {
		contents = ReadValue(jsonResponse, []interface{}{
			"continuationContents", "playlistVideoListContinuation", "contents",
		})

		nextContinuation = ReadValueString(jsonResponse, []interface{}{
			"continuationContents", "playlistVideoListContinuation",
			"continuations", 0, "nextContinuationData", "continuation",
		})
	}

	tracks := []Track{}

	if contentsList, ok := contents.([]interface{}); ok {
		for _, item := range contentsList {
			track := parseTrackItem(item)
			if track.VideoId != "" {
				tracks = append(tracks, track)
			}
		}
	}

	return tracks, nextContinuation
}

// ParseUserPlaylists extracts user playlists from the library response
func (p *TVHTML5Parser) ParseUserPlaylists(jsonResponse interface{}, isFirstPage bool) ([]YoutubePlaylist, string) {
	var playlistItems interface{}
	var nextContinuation string

	if isFirstPage {
		playlistItems = ReadValue(jsonResponse, []interface{}{
			"contents", "tvBrowseRenderer", "content", "tvSecondaryNavRenderer",
			"sections", 0, "tvSecondaryNavSectionRenderer", "tabs", 1,
			"tabRenderer", "content", "tvSurfaceContentRenderer", "content",
			"gridRenderer", "items",
		})

		nextContinuation = ReadValueString(jsonResponse, []interface{}{
			"contents", "tvBrowseRenderer", "content", "tvSecondaryNavRenderer",
			"sections", 0, "tvSecondaryNavSectionRenderer", "tabs", 1,
			"tabRenderer", "content", "tvSurfaceContentRenderer",
			"continuation", "reloadContinuationData", "continuation",
		})
	} else {
		playlistItems = ReadValue(jsonResponse, []interface{}{
			"continuationContents", "gridRenderer", "items",
		})

		nextContinuation = ReadValueString(jsonResponse, []interface{}{
			"continuationContents", "gridRenderer", "continuation",
			"reloadContinuationData", "continuation",
		})
	}

	youtubePlaylists := []YoutubePlaylist{}

	if items, ok := playlistItems.([]interface{}); ok {
		for _, item := range items {
			playlist := parseTVHTML5PlaylistItem(item)
			if playlist.PlaylistId != "" {
				youtubePlaylists = append(youtubePlaylists, playlist)
			}
		}
	}

	return youtubePlaylists, nextContinuation
}

func parseTVHTML5PlaylistItem(item interface{}) YoutubePlaylist {
	tileRenderer := ReadValue(item, []interface{}{"tileRenderer"})
	if tileRenderer == nil {
		return YoutubePlaylist{}
	}

	title := ReadValueString(tileRenderer, []interface{}{
		"metadata", "tileMetadataRenderer", "title", "runs", 0, "text",
	})

	browseId := ReadValueString(tileRenderer, []interface{}{
		"onSelectCommand", "browseEndpoint", "browseId",
	})

	if browseId == "" {
		browseId = ReadValueString(tileRenderer, []interface{}{
			"metadata", "tileMetadataRenderer", "title", "runs", 0,
			"navigationEndpoint", "browseEndpoint", "browseId",
		})
	}

	playlistId := browseId
	if len(playlistId) > 2 && strings.HasPrefix(playlistId, "VL") {
		playlistId = playlistId[2:]
	}

	thumbnails := ReadValue(tileRenderer, []interface{}{
		"header", "tileHeaderRenderer", "thumbnail", "thumbnails",
	})

	thumbnailUrls := []string{}
	if thumbsList, ok := thumbnails.([]interface{}); ok {
		for _, thumb := range thumbsList {
			if thumbMap, ok := thumb.(map[string]interface{}); ok {
				if url, ok := thumbMap["url"].(string); ok && url != "" {
					thumbnailUrls = append(thumbnailUrls, url)
				}
			}
		}
	}

	totalTracks := ""
	metadataText := ReadValueString(tileRenderer, []interface{}{
		"metadata", "tileMetadataRenderer", "lines", 1,
		"lineRenderer", "items", 0, "lineItemRenderer", "text", "simpleText",
	})

	if metadataText != "" {
		totalTracks = extractTrackCountFromMetadata(metadataText)
	}

	return YoutubePlaylist{
		Title:       title,
		Thumbnails:  thumbnailUrls,
		TotalTracks: totalTracks,
		PlaylistId:  playlistId,
		Url:         CreatePlaylistLink(playlistId),
	}
}

func parseFeaturedArtists(title string) (string, []string) {
	titleWithFeaturedArtists := tvhtml5TrackTitleWithFeatRegex.FindStringSubmatch(title)

	if len(titleWithFeaturedArtists) > 2 {
		cleanTitle := titleWithFeaturedArtists[1]
		featuredArtist := titleWithFeaturedArtists[2]
		artists := strings.Split(featuredArtist, fmt.Sprintf(" %s ", AND_SEPARATOR))
		return cleanTitle, artists
	}

	return title, []string{}
}

func cleanTrackCount(totalTracks string) int {
	totalTracks = strings.TrimSpace(totalTracks)
	totalTracks = strings.Replace(totalTracks, " tracks", "", 1)
	totalTracks = strings.Replace(totalTracks, " songs", "", 1)
	totalTracks = strings.ReplaceAll(totalTracks, ",", "")

	count, _ := strconv.Atoi(totalTracks)
	return count
}

func extractTrackCountFromMetadata(metadataText string) string {
	parts := strings.Split(metadataText, "•")
	if len(parts) < 2 {
		parts = strings.Split(metadataText, "·")
	}

	if len(parts) >= 2 {
		countPart := strings.TrimSpace(parts[len(parts)-1])
		fields := strings.Fields(countPart)
		if len(fields) > 0 {
			return fields[0]
		}
	}

	return ""
}
