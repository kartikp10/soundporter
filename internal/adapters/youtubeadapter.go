package adapters

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"soundporter/internal/playlist"
	"soundporter/internal/utils"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const youtubeRedirectURI = "http://localhost:8080/callback"

// YouTubeAdapter adapts the YouTube API to our common adapter interface
type YouTubeAdapter struct {
	BaseAdapter
	service      *youtube.Service
	clientID     string
	clientSecret string
	ch           chan *youtube.Service
	state        string
}

// NewYouTubeAdapter creates a new YouTubeAdapter
func NewYouTubeAdapter(clientID, clientSecret string) (*YouTubeAdapter, error) {
	if clientID == "" {
		clientID = os.Getenv("YOUTUBE_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("YOUTUBE_CLIENT_SECRET")
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("youtube client ID and secret must be provided or set in environment variables")
	}

	return &YouTubeAdapter{
		BaseAdapter:  NewBaseAdapter("YouTube"),
		clientID:     clientID,
		clientSecret: clientSecret,
		ch:           make(chan *youtube.Service),
		state:        utils.GenerateState(),
	}, nil
}

// Authenticate handles user authentication with YouTube API
func (a *YouTubeAdapter) Authenticate() error {
	// OAuth2 config for YouTube API
	config := &oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		RedirectURL:  youtubeRedirectURI,
		Scopes: []string{
			youtube.YoutubeReadonlyScope,
			youtube.YoutubeScope,
		},
		Endpoint: google.Endpoint,
	}

	// Start HTTP server for OAuth callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		a.completeAuth(w, r, config)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})

	// Start the HTTP server
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Generate the authorization URL
	authURL := config.AuthCodeURL(a.state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Please log in to YouTube by visiting the following page in your browser:", authURL)

	// Open the URL in the user's browser
	utils.OpenBrowser(authURL)

	// Wait for auth to complete
	a.service = <-a.ch
	a.SetAuthenticated(true)

	fmt.Println("YouTube authentication successful!")
	return nil
}

// GetUserPlaylists retrieves all playlists for the authenticated user
func (a *YouTubeAdapter) GetUserPlaylists() ([]playlist.Playlist, error) {
	if err := a.CheckAuth(); err != nil {
		return nil, err
	}

	var playlists []playlist.Playlist
	var nextPageToken string

	for {
		call := a.service.Playlists.List([]string{"snippet", "contentDetails"}).
			Mine(true).
			MaxResults(50)

		if nextPageToken != "" {
			call = call.PageToken(nextPageToken)
		}

		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("error fetching playlists: %v", err)
		}

		for _, item := range response.Items {
			publishedTime, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			playlists = append(playlists, playlist.Playlist{
				ID:          item.Id,
				Name:        item.Snippet.Title,
				Description: item.Snippet.Description,
				TrackCount:  int(item.ContentDetails.ItemCount),
				CreatedAt:   publishedTime,
			})
		}

		nextPageToken = response.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	return playlists, nil
}

// GetPlaylistItems retrieves all tracks (videos) in a playlist
func (a *YouTubeAdapter) GetPlaylistItems(playlistID string) ([]playlist.Track, error) {
	if err := a.CheckAuth(); err != nil {
		return nil, err
	}

	var tracks []playlist.Track
	var nextPageToken string

	for {
		call := a.service.PlaylistItems.List([]string{"snippet", "contentDetails"}).
			PlaylistId(playlistID).
			MaxResults(50)

		if nextPageToken != "" {
			call = call.PageToken(nextPageToken)
		}

		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("error fetching playlist items: %v", err)
		}

		for _, item := range response.Items {
			videoID := item.ContentDetails.VideoId
			videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

			track := playlist.Track{
				Name:      item.Snippet.Title,
				Artists:   []string{item.Snippet.VideoOwnerChannelTitle},
				ID:        videoID,
				ArtistIDs: []string{item.Snippet.VideoOwnerChannelId},
				URL:       videoURL,
			}

			tracks = append(tracks, track)
		}

		nextPageToken = response.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	return tracks, nil
}

// CreateNewPlaylist creates a new YouTube playlist
func (a *YouTubeAdapter) CreateNewPlaylist(name string, description string) (playlist.Playlist, error) {
	if err := a.CheckAuth(); err != nil {
		return playlist.Playlist{}, err
	}

	p := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title:       name,
			Description: description,
		},
		Status: &youtube.PlaylistStatus{
			PrivacyStatus: "private",
		},
	}

	response, err := a.service.Playlists.Insert([]string{"snippet", "status"}, p).Do()
	if err != nil {
		return playlist.Playlist{}, fmt.Errorf("error creating playlist: %v", err)
	}
	publishedTime, _ := time.Parse(time.RFC3339, response.Snippet.PublishedAt)
	return playlist.Playlist{
		ID:          response.Id,
		Name:        response.Snippet.Title,
		Description: response.Snippet.Description,
		TrackCount:  0,
		CreatedAt:   publishedTime,
	}, nil
}

// AddItemsToPlaylist adds videos to a YouTube playlist
func (a *YouTubeAdapter) AddItemsToPlaylist(playlistID string, trackIDs []string) error {
	if err := a.CheckAuth(); err != nil {
		return err
	}

	for _, videoID := range trackIDs {
		// Extract video ID if it's a URL
		if strings.Contains(videoID, "youtube.com/watch?v=") {
			parts := strings.Split(videoID, "v=")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "&")[0]
			}
		} else if strings.Contains(videoID, "youtu.be/") {
			parts := strings.Split(videoID, "youtu.be/")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "?")[0]
			}
		}

		// Add video to playlist
		playlistItem := &youtube.PlaylistItem{
			Snippet: &youtube.PlaylistItemSnippet{
				PlaylistId: playlistID,
				ResourceId: &youtube.ResourceId{
					Kind:    "youtube#video",
					VideoId: videoID,
				},
			},
		}

		_, err := a.service.PlaylistItems.Insert([]string{"snippet"}, playlistItem).Do()
		if err != nil {
			return fmt.Errorf("error adding video %s to playlist: %v", videoID, err)
		}

		// YouTube API has quota limits, so add a small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// SearchTracks searches for videos on YouTube
func (a *YouTubeAdapter) SearchTracks(query string, limit int) ([]playlist.Track, error) {
	if err := a.CheckAuth(); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 50 {
		limit = 50 // YouTube API maximum is 50 per request
	}

	response, err := a.service.Search.List([]string{"snippet"}).
		Q(query).
		Type("video").
		MaxResults(int64(limit)).
		Do()

	if err != nil {
		return nil, fmt.Errorf("error searching for videos: %v", err)
	}

	var tracks []playlist.Track

	for _, item := range response.Items {
		videoID := item.Id.VideoId
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

		tracks = append(tracks, playlist.Track{
			Name:      item.Snippet.Title,
			Artists:   []string{item.Snippet.ChannelTitle},
			ID:        videoID,
			ArtistIDs: []string{item.Snippet.ChannelId},
			URL:       videoURL,
		})
	}

	return tracks, nil
}

// completeAuth handles the OAuth2 callback
func (a *YouTubeAdapter) completeAuth(w http.ResponseWriter, r *http.Request, config *oauth2.Config) {
	if r.FormValue("state") != a.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", r.FormValue("state"), a.state)
	}

	code := r.FormValue("code")
	token, err := config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusInternalServerError)
		log.Fatalf("Error exchanging code for token: %v", err)
	}

	client := config.Client(r.Context(), token)
	service, err := youtube.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "Error creating YouTube client", http.StatusInternalServerError)
		log.Fatalf("Error creating YouTube client: %v", err)
	}

	fmt.Fprintf(w, "Login Completed! You can now close this window.")
	a.ch <- service
}
