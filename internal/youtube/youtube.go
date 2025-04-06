// 1. Register an application at: https://console.developers.google.com/
//   - Enable the YouTube Data API v3
//   - Create OAuth 2.0 credentials (Other/Desktop application type)
//
// 2. Set the YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET environment variables.

package youtubePorter

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"soundporter/internal/utils"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const redirectURI = "http://localhost:8080/callback"

var (
	ch    = make(chan *youtube.Service)
	state = utils.GenerateState()
)

type YouTubePorter struct {
	Service *youtube.Service
}

// Auth authenticates with YouTube API
func (p *YouTubePorter) Auth() error {
	// OAuth2 config for YouTube API
	config := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  redirectURI,
		Scopes: []string{
			youtube.YoutubeReadonlyScope,
			youtube.YoutubeScope,
		},
		Endpoint: google.Endpoint,
	}

	// Start HTTP server for OAuth callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		completeAuth(w, r, config)
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
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Please log in to YouTube by visiting the following page in your browser:", authURL)

	// Open the URL in the user's browser
	utils.OpenBrowser(authURL)

	// Wait for auth to complete
	service := <-ch

	p.Service = service
	fmt.Println("YouTube authentication successful!")
	return nil
}

// Export exports a YouTube playlist to a CSV file
func (p *YouTubePorter) Export() error {
	if p.Service == nil {
		return fmt.Errorf("not authenticated, call Auth() first")
	}

	reader := bufio.NewReader(os.Stdin)

	// Get the list of user's playlists
	fmt.Println("Fetching your playlists...")
	response, err := p.Service.Playlists.List([]string{"snippet", "contentDetails"}).Mine(true).MaxResults(50).Do()
	if err != nil {
		return fmt.Errorf("error fetching playlists: %v", err)
	}

	// Display available playlists
	fmt.Println("\nYour playlists:")
	for i, playlist := range response.Items {
		fmt.Printf("%d. %s (%d videos)\n", i+1, playlist.Snippet.Title, playlist.ContentDetails.ItemCount)
	}

	// Get user selection
	fmt.Print("\nEnter the number of the playlist to export: ")
	var playlistIndex int
	fmt.Scanln(&playlistIndex)
	playlistIndex--

	if playlistIndex < 0 || playlistIndex >= len(response.Items) {
		return fmt.Errorf("invalid playlist selection")
	}

	selectedPlaylist := response.Items[playlistIndex]

	// Get output filename
	fmt.Print("Enter output filename (default: playlist.csv): ")
	fileName, _ := reader.ReadString('\n')
	fileName = strings.TrimSpace(fileName)

	if fileName == "" {
		fileName = "playlist.csv"
	} else if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}

	// Create CSV file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Video Title", "Channel Name", "Video ID", "Channel ID", "Video URL"}
	writer.Write(header)

	// Fetch playlist items
	var nextPageToken string
	count := 0
	fmt.Println("Fetching playlist items...")

	for {
		playlistItemsCall := p.Service.PlaylistItems.List([]string{"snippet", "contentDetails"}).
			PlaylistId(selectedPlaylist.Id).
			MaxResults(50)

		if nextPageToken != "" {
			playlistItemsCall = playlistItemsCall.PageToken(nextPageToken)
		}

		playlistItemsResponse, err := playlistItemsCall.Do()
		if err != nil {
			return fmt.Errorf("error fetching playlist items: %v", err)
		}

		// Process each video in the playlist
		for _, item := range playlistItemsResponse.Items {
			snippet := item.Snippet
			videoID := item.ContentDetails.VideoId
			videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

			row := []string{
				snippet.Title,
				snippet.VideoOwnerChannelTitle,
				videoID,
				snippet.VideoOwnerChannelId,
				videoURL,
			}

			writer.Write(row)
			count++
		}

		// Check if there are more pages
		nextPageToken = playlistItemsResponse.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	fmt.Printf("Successfully exported %d videos from playlist '%s' to %s\n",
		count, selectedPlaylist.Snippet.Title, fileName)
	return nil
}

// Import imports videos from a CSV file to a new YouTube playlist
func (p *YouTubePorter) Import() error {
	if p.Service == nil {
		return fmt.Errorf("not authenticated, call Auth() first")
	}

	reader := bufio.NewReader(os.Stdin)

	// Get CSV file path
	fmt.Print("Enter CSV file path to import: ")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

	// Get new playlist name
	fmt.Print("Enter new playlist name: ")
	playlistName, _ := reader.ReadString('\n')
	playlistName = strings.TrimSpace(playlistName)

	if playlistName == "" {
		playlistName = "Imported Playlist"
	}

	// Open and read CSV file
	csvFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening CSV file: %v", err)
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV file: %v", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or contains only header")
	}

	// Find the video ID column index
	videoIDIndex := -1
	header := records[0]

	for i, colName := range header {
		colNameLower := strings.ToLower(colName)
		if strings.Contains(colNameLower, "video id") ||
			strings.Contains(colNameLower, "videoid") ||
			strings.Contains(colNameLower, "track id") {
			videoIDIndex = i
			break
		}
	}

	if videoIDIndex == -1 {
		return fmt.Errorf("video ID column not found in CSV")
	}

	// Create a new playlist
	playlistDescription := fmt.Sprintf("Imported playlist using Soundporter on %s", time.Now().Format("2006-01-02"))

	playlist := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title:       playlistName,
			Description: playlistDescription,
		},
		Status: &youtube.PlaylistStatus{
			PrivacyStatus: "private",
		},
	}

	playlistResponse, err := p.Service.Playlists.Insert([]string{"snippet", "status"}, playlist).Do()
	if err != nil {
		return fmt.Errorf("error creating playlist: %v", err)
	}

	fmt.Printf("Created playlist: %s (ID: %s)\n", playlistName, playlistResponse.Id)

	// Add videos to playlist
	var skippedVideos int
	var addedVideos int

	for _, record := range records[1:] {
		if len(record) <= videoIDIndex || record[videoIDIndex] == "" {
			skippedVideos++
			continue
		}

		videoID := record[videoIDIndex]

		// For Spotify tracks, try to search for the equivalent YouTube video
		if !strings.HasPrefix(videoID, "http") && len(videoID) < 15 {
			// This is likely a Spotify ID, search for the video
			searchQuery := ""
			if len(record) > 0 && record[0] != "" {
				// Use track name for search
				searchQuery = record[0]
				if len(record) > 1 && record[1] != "" {
					// Add artist name for better search
					searchQuery += " " + record[1]
				}

				// Search for the video on YouTube
				searchResponse, err := p.Service.Search.List([]string{"id"}).
					Q(searchQuery).
					Type("video").
					MaxResults(1).
					Do()

				if err != nil {
					fmt.Printf("Error searching for video '%s': %v\n", searchQuery, err)
					skippedVideos++
					continue
				}

				if len(searchResponse.Items) == 0 {
					fmt.Printf("No videos found for '%s'\n", searchQuery)
					skippedVideos++
					continue
				}

				videoID = searchResponse.Items[0].Id.VideoId
			} else {
				skippedVideos++
				continue
			}
		} else if strings.Contains(videoID, "youtube.com/watch?v=") {
			// Extract video ID from URL
			parts := strings.Split(videoID, "v=")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "&")[0]
			}
		} else if strings.Contains(videoID, "youtu.be/") {
			// Extract video ID from shortened URL
			parts := strings.Split(videoID, "youtu.be/")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "?")[0]
			}
		}

		// Add video to playlist
		playlistItem := &youtube.PlaylistItem{
			Snippet: &youtube.PlaylistItemSnippet{
				PlaylistId: playlistResponse.Id,
				ResourceId: &youtube.ResourceId{
					Kind:    "youtube#video",
					VideoId: videoID,
				},
			},
		}

		_, err := p.Service.PlaylistItems.Insert([]string{"snippet"}, playlistItem).Do()
		if err != nil {
			fmt.Printf("Error adding video %s: %v\n", videoID, err)
			skippedVideos++
			continue
		}

		addedVideos++

		// YouTube API has quota limits, so add a small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("Successfully created playlist '%s' with %d videos\n", playlistName, addedVideos)
	if skippedVideos > 0 {
		fmt.Printf("Skipped %d videos due to missing or invalid video IDs\n", skippedVideos)
	}

	playlistURL := fmt.Sprintf("https://www.youtube.com/playlist?list=%s", playlistResponse.Id)
	fmt.Printf("Playlist URL: %s\n", playlistURL)

	return nil
}

// completeAuth handles the OAuth2 callback
func completeAuth(w http.ResponseWriter, r *http.Request, config *oauth2.Config) {
	if r.FormValue("state") != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", r.FormValue("state"), state)
	}

	code := r.FormValue("code")
	token, err := config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusInternalServerError)
		log.Fatalf("Error exchanging code for token: %v", err)
	}

	client := config.Client(r.Context(), token)
	service, err := youtube.NewService(r.Context(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "Error creating YouTube client", http.StatusInternalServerError)
		log.Fatalf("Error creating YouTube client: %v", err)
	}

	fmt.Fprintf(w, "Login Completed! You can now close this window.")
	ch <- service
}
