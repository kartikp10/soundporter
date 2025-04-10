package adapters

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"soundporter/internal/playlist"
	"soundporter/internal/utils"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const spotifyRedirectURI = "http://localhost:8080/callback"

// SpotifyAdapter adapts the Spotify API to our common adapter interface
type SpotifyAdapter struct {
	BaseAdapter  // Embed the BaseAdapter
	client       *spotify.Client
	clientID     string
	clientSecret string
	ch           chan *spotify.Client
	state        string
}

// NewSpotifyAdapter creates a new SpotifyAdapter
func NewSpotifyAdapter(clientID, clientSecret string) (*SpotifyAdapter, error) {
	if clientID == "" {
		clientID = os.Getenv("SPOTIFY_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("SPOTIFY_SECRET")
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("spotify client ID and secret must be provided or set in environment variables")
	}

	return &SpotifyAdapter{
		BaseAdapter:  NewBaseAdapter("Spotify"),
		clientID:     clientID,
		clientSecret: clientSecret,
		ch:           make(chan *spotify.Client),
		state:        utils.GenerateState(),
	}, nil
}

// Authenticate handles user authentication with Spotify
func (a *SpotifyAdapter) Authenticate() error {
	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(spotifyRedirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopePlaylistModifyPrivate),
		spotifyauth.WithClientID(a.clientID),
		spotifyauth.WithClientSecret(a.clientSecret),
	)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		a.completeAuth(w, r, auth)
	})

	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Open browser for authentication
	url := auth.AuthURL(a.state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
	utils.OpenBrowser(url)

	// Wait for authentication to complete
	a.client = <-a.ch
	a.SetAuthenticated(true)

	// Verify authentication by getting user info
	user, err := a.client.CurrentUser(context.Background())
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	fmt.Println("You are logged in as:", user.ID)
	return nil
}

// GetUserPlaylists retrieves all playlists for the authenticated user
func (a *SpotifyAdapter) GetUserPlaylists() ([]playlist.Playlist, error) {
	if err := a.CheckAuth(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	var allPlaylists []playlist.Playlist
	limit := 50
	offset := 0

	for {
		playlistPage, err := a.client.CurrentUsersPlaylists(ctx, spotify.Limit(limit), spotify.Offset(offset))
		if err != nil {
			return nil, fmt.Errorf("error getting playlists: %v", err)
		}

		for _, p := range playlistPage.Playlists {
			allPlaylists = append(allPlaylists, playlist.Playlist{
				ID:         string(p.ID),
				Name:       p.Name,
				TrackCount: int(p.Tracks.Total),
				CreatedAt:  time.Now(), // Spotify doesn't provide creation date easily
			})
		}

		if len(playlistPage.Playlists) < limit {
			break
		}
		offset += limit
	}

	return allPlaylists, nil
}

// GetPlaylistItems retrieves all tracks in a playlist
func (a *SpotifyAdapter) GetPlaylistItems(playlistID string) ([]playlist.Track, error) {
	if err := a.CheckAuth(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	var tracks []playlist.Track
	limit := 50
	offset := 0

	for {
		playlistItems, err := a.client.GetPlaylistItems(
			ctx,
			spotify.ID(playlistID),
			spotify.Limit(limit),
			spotify.Offset(offset),
		)
		if err != nil {
			return nil, fmt.Errorf("error getting playlist items: %v", err)
		}

		for _, item := range playlistItems.Items {
			track := item.Track.Track

			// Convert artists
			var artistNames []string
			var artistIDs []string
			for _, artist := range track.Artists {
				artistNames = append(artistNames, artist.Name)
				artistIDs = append(artistIDs, string(artist.ID))
			}

			tracks = append(tracks, playlist.Track{
				Name:      track.Name,
				Artists:   artistNames,
				Album:     track.Album.Name,
				ID:        string(track.ID),
				ArtistIDs: artistIDs,
				AlbumID:   string(track.Album.ID),
				URL:       fmt.Sprintf("https://open.spotify.com/track/%s", track.ID),
			})
		}

		if len(playlistItems.Items) < limit {
			break
		}
		offset += limit
	}

	return tracks, nil
}

// CreateNewPlaylist creates a new Spotify playlist
func (a *SpotifyAdapter) CreateNewPlaylist(name string, description string) (playlist.Playlist, error) {
	if err := a.CheckAuth(); err != nil {
		return playlist.Playlist{}, err
	}

	ctx := context.Background()
	user, err := a.client.CurrentUser(ctx)
	if err != nil {
		return playlist.Playlist{}, fmt.Errorf("error getting current user: %v", err)
	}

	p, err := a.client.CreatePlaylistForUser(
		ctx,
		user.ID,
		name,
		description,
		false, // Not public
		false, // Not collaborative
	)
	if err != nil {
		return playlist.Playlist{}, fmt.Errorf("error creating playlist: %v", err)
	}

	return playlist.Playlist{
		ID:          string(p.ID),
		Name:        p.Name,
		Description: p.Description,
		TrackCount:  0,
		CreatedAt:   time.Now(),
	}, nil
}

// AddItemsToPlaylist adds tracks to a Spotify playlist
func (a *SpotifyAdapter) AddItemsToPlaylist(playlistID string, trackIDs []string) error {
	if err := a.CheckAuth(); err != nil {
		return err
	}

	// Convert string IDs to Spotify IDs
	var spotifyTrackIDs []spotify.ID
	for _, id := range trackIDs {
		spotifyTrackIDs = append(spotifyTrackIDs, spotify.ID(id))
	}

	ctx := context.Background()
	_, err := a.client.AddTracksToPlaylist(ctx, spotify.ID(playlistID), spotifyTrackIDs...)
	if err != nil {
		return fmt.Errorf("error adding tracks to playlist: %v", err)
	}

	return nil
}

// SearchTracks searches for tracks on Spotify
func (a *SpotifyAdapter) SearchTracks(query string, limit int) ([]playlist.Track, error) {
	if err := a.CheckAuth(); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 50 {
		limit = 50 // Spotify API maximum is 50 per request
	}

	ctx := context.Background()
	results, err := a.client.Search(
		ctx,
		query,
		spotify.SearchTypeTrack,
		spotify.Limit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching tracks: %v", err)
	}

	var tracks []playlist.Track
	for _, item := range results.Tracks.Tracks {
		// Convert artists
		var artistNames []string
		var artistIDs []string
		for _, artist := range item.Artists {
			artistNames = append(artistNames, artist.Name)
			artistIDs = append(artistIDs, string(artist.ID))
		}

		tracks = append(tracks, playlist.Track{
			Name:      item.Name,
			Artists:   artistNames,
			Album:     item.Album.Name,
			ID:        string(item.ID),
			ArtistIDs: artistIDs,
			AlbumID:   string(item.Album.ID),
			URL:       fmt.Sprintf("https://open.spotify.com/track/%s", item.ID),
		})
	}

	return tracks, nil
}

// completeAuth is the callback handler for the Spotify auth flow
func (a *SpotifyAdapter) completeAuth(w http.ResponseWriter, r *http.Request, auth *spotifyauth.Authenticator) {
	tok, err := auth.Token(r.Context(), a.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != a.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, a.state)
	}

	// Use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed! You can now close this window.")
	a.ch <- client
}
