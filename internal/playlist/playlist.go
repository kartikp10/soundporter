package playlist

import "time"

// Track represents a single music track with essential metadata
type Track struct {
	Name      string   `csv:"name"`
	Artists   []string `csv:"artists"`
	Album     string   `csv:"album"`
	ID        string   `csv:"id"`
	ArtistIDs []string `csv:"artist_ids"`
	AlbumID   string   `csv:"album_id"`
	URL       string   `csv:"url"`
}

// Playlist represents a collection of tracks
type Playlist struct {
	ID          string
	Name        string
	Description string
	TrackCount  int
	Tracks      []Track
	CreatedAt   time.Time
}
