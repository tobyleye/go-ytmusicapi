package parsers

// Track represents a YouTube Music track
type Track struct {
	VideoId string
	Title   string
	Artists []string
	Link    string
}

// YoutubePlaylist represents a YouTube Music playlist
type YoutubePlaylist struct {
	Title       string   `json:"title"`
	Thumbnails  []string `json:"thumbnails"`
	TotalTracks string   `json:"total_tracks"`
	PlaylistId  string   `json:"playlist_id"`
	Url         string   `json:"url"`
}

// PlaylistDetails represents detailed playlist information
type PlaylistDetails struct {
	Title          string
	Description    string
	TotalTracks    int
	PlaylistTracks []Track
	Link           string
	Thumbnails     []string
}

// PlaylistAllTracksResponse represents all tracks in a playlist
type PlaylistAllTracksResponse struct {
	Total  int     `json:"total"`
	Tracks []Track `json:"tracks"`
}

// PlaylistTracksResponse represents a paginated response of playlist tracks
type PlaylistTracksResponse struct {
	NextContinuation string  `json:"next_continuation"`
	Tracks           []Track `json:"tracks"`
}

// CreatedPlaylist represents a newly created playlist
type CreatedPlaylist struct {
	PlaylistId  string `json:"playlist_id"`
	Link        string `json:"link"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// PlaylistPageResponse represents a paginated response of playlists
type PlaylistPageResponse struct {
	Continuation string            `json:"continuation"`
	Playlists    []YoutubePlaylist `json:"playlists"`
}
