package parsers

// Parser defines the interface for YouTube Music API response parsers
// Different parsers implement this interface for different client types (WEB_REMIX, TVHTML5, etc.)
type Parser interface {
	// ParseSearchResults extracts tracks from a search response
	ParseSearchResults(data interface{}) []Track

	// ParsePlaylistDetails extracts playlist metadata and tracks from a playlist fetch response
	ParsePlaylistDetails(jsonResponse interface{}) PlaylistDetails

	// ParsePlaylistTracks extracts tracks and continuation token from playlist tracks response
	// isFirstPage indicates whether this is the first page or a continuation page
	ParsePlaylistTracks(jsonResponse interface{}, isFirstPage bool) ([]Track, string)

	// ParseUserPlaylists extracts user playlists from the library response
	// isFirstPage indicates whether this is the first page or a continuation page
	ParseUserPlaylists(jsonResponse interface{}, isFirstPage bool) ([]YoutubePlaylist, string)
}
