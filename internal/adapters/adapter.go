package adapters

import (
	"fmt"
	"soundporter/internal/playlist"
)

// ApiAdapter defines the interface for adapting different music platform APIs
// to a common interface that can be used by the application
type ApiAdapter interface {
	// Authentication methods
	Authenticate() error
	IsAuthenticated() bool

	// Platform-specific methods
	GetUserPlaylists() ([]playlist.Playlist, error)
	GetPlaylistItems(playlistID string) ([]playlist.Track, error)
	CreateNewPlaylist(name string, description string) (playlist.Playlist, error)
	AddItemsToPlaylist(playlistID string, trackIDs []string) error

	// Search functionality
	SearchTracks(query string, limit int) ([]playlist.Track, error)
}

// PlatformType represents the supported music platforms
type PlatformType string

const (
	SpotifyPlatform    PlatformType = "spotify"
	YoutubePlatform    PlatformType = "youtube"
	AppleMusicPlatform PlatformType = "apple"
)

// NewApiAdapter is a factory function that creates a new adapter for the specified platform
func NewApiAdapter(platform string, clientID, clientSecret string) (ApiAdapter, error) {
	p := PlatformType(platform)
	switch p {
	case SpotifyPlatform:
		return NewSpotifyAdapter(clientID, clientSecret)
	case YoutubePlatform:
		return NewYouTubeAdapter(clientID, clientSecret)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}
