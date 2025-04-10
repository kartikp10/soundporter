package porter

import (
	"encoding/csv"
	"fmt"
	"os"
	"soundporter/internal/adapters"
	"soundporter/internal/playlist"
	"strings"
	"time"
)

// Porter implements the core.Porter interface
// using an adapter to interact with a specific music platform
type Porter struct {
	adapter adapters.ApiAdapter
}

// NewPorter creates a new playlist service using the specified adapter
func NewPorter(adapter adapters.ApiAdapter) *Porter {
	return &Porter{
		adapter: adapter,
	}
}

// NewPorterWithPlatform creates a new Porter instance with the specified platform type
func NewPorterWithCredentials(platform string, clientID, clientSecret string) (*Porter, error) {
	adapter, err := adapters.NewApiAdapter(platform, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter for platform %s: %v", platform, err)
	}
	return NewPorter(adapter), nil
}

// Authenticate delegates authentication to the adapter
func (s *Porter) Authenticate() error {
	return s.adapter.Authenticate()
}

// IsAuthenticated checks if the service is authenticated
func (s *Porter) IsAuthenticated() bool {
	return s.adapter.IsAuthenticated()
}

// GetPlaylists retrieves all playlists via the adapter
func (s *Porter) GetPlaylists() ([]playlist.Playlist, error) {
	return s.adapter.GetUserPlaylists()
}

// GetPlaylistTracks retrieves all tracks in a playlist
func (s *Porter) GetPlaylistTracks(playlistID string) ([]playlist.Track, error) {
	return s.adapter.GetPlaylistItems(playlistID)
}

// CreatePlaylist creates a new playlist
func (s *Porter) CreatePlaylist(name, description string) (playlist.Playlist, error) {
	if description == "" {
		description = fmt.Sprintf("Playlist created via Soundporter on %s", time.Now().Format("2006-01-02"))
	}
	return s.adapter.CreateNewPlaylist(name, description)
}

// AddTracksToPlaylist adds tracks to an existing playlist
func (s *Porter) AddTracksToPlaylist(playlistID string, trackIDs []string) error {
	return s.adapter.AddItemsToPlaylist(playlistID, trackIDs)
}

// ExportPlaylistToCSV exports a playlist to a CSV file
func (s *Porter) ExportPlaylistToCSV(playlistID, filepath string) error {
	// Get tracks from playlist
	tracks, err := s.adapter.GetPlaylistItems(playlistID)
	if err != nil {
		return fmt.Errorf("failed to get playlist tracks: %v", err)
	}

	// Ensure filepath has .csv extension
	if !strings.HasSuffix(filepath, ".csv") {
		filepath += ".csv"
	}

	// Create CSV file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %v", err)
	}
	defer file.Close()

	// Write tracks to CSV
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Track Name", "Artist Name", "Album Name", "Track ID", "Artist ID", "Album ID", "Track URL"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing CSV header: %v", err)
	}

	// Write track data
	for _, track := range tracks {
		artistNames := strings.Join(track.Artists, ", ")
		artistIDs := strings.Join(track.ArtistIDs, ", ")

		row := []string{
			track.Name,
			artistNames,
			track.Album,
			track.ID,
			artistIDs,
			track.AlbumID,
			track.URL,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing track to CSV: %v", err)
		}
	}

	return nil
}

// ImportPlaylistFromCSV imports a playlist from a CSV file
func (s *Porter) ImportPlaylistFromCSV(filepath string, playlistName string) error {
	// Open and read CSV file
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("error opening CSV file: %v", err)
	}
	defer file.Close()

	// Read CSV data
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV file: %v", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or contains only header")
	}

	// Find the track ID column index
	header := records[0]
	trackIDIndex := -1
	for i, colName := range header {
		if strings.Contains(strings.ToLower(colName), "track id") {
			trackIDIndex = i
			break
		}
	}

	if trackIDIndex == -1 {
		return fmt.Errorf("track ID column not found in CSV")
	}

	// Create a new playlist
	description := fmt.Sprintf("Playlist imported via Soundporter on %s", time.Now().Format("2006-01-02"))
	playlist, err := s.adapter.CreateNewPlaylist(playlistName, description)
	if err != nil {
		return fmt.Errorf("error creating playlist: %v", err)
	}

	// Extract track IDs
	var trackIDs []string
	var skippedTracks int

	for _, record := range records[1:] {
		if len(record) <= trackIDIndex || record[trackIDIndex] == "" {
			skippedTracks++
			continue
		}

		trackIDs = append(trackIDs, record[trackIDIndex])

		// Add tracks in batches of 100 (Spotify's limit)
		if len(trackIDs) >= 100 {
			if err := s.adapter.AddItemsToPlaylist(playlist.ID, trackIDs); err != nil {
				return fmt.Errorf("error adding tracks to playlist: %v", err)
			}
			trackIDs = nil
		}
	}

	// Add remaining tracks
	if len(trackIDs) > 0 {
		if err := s.adapter.AddItemsToPlaylist(playlist.ID, trackIDs); err != nil {
			return fmt.Errorf("error adding remaining tracks to playlist: %v", err)
		}
	}

	fmt.Printf("Successfully imported playlist '%s' with %d tracks\n", playlistName, len(records)-1-skippedTracks)
	if skippedTracks > 0 {
		fmt.Printf("Skipped %d tracks due to missing or invalid track IDs\n", skippedTracks)
	}

	return nil
}
