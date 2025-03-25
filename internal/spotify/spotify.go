// 1. Register an application at: https://developer.spotify.com/my-applications/
//   - Use "http://localhost:8080/callback" as the redirect URI
//
// 2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
// 3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package spotifyPorter

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate))
	ch    = make(chan *spotify.Client)
	state = generateState()
)

type SpotifyPorter struct {
	Client *spotify.Client
}

func (p *SpotifyPorter) Auth() error {
	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	p.Client = <-ch

	// use the client to make calls that require authorization
	user, err := p.Client.CurrentUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("You are logged in as:", user.ID)
	return nil
}

func (p *SpotifyPorter) Export() error {
	if p.Client == nil {
		return fmt.Errorf("not authenticated, call Auth() first")
	}

	ctx := context.Background()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter playlist name: ")
	playlistName, _ := reader.ReadString('\n')
	playlistName = strings.TrimSpace(playlistName)

	fmt.Print("Enter output filename (default: playlist.csv): ")
	fileName, _ := reader.ReadString('\n')
	fileName = strings.TrimSpace(fileName)

	if fileName == "" {
		fileName = "playlist.csv"
	} else if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}

	// Get user's playlists
	playlists, err := p.Client.CurrentUsersPlaylists(ctx)
	if err != nil {
		return fmt.Errorf("error getting playlists: %v", err)
	}

	var playlistID spotify.ID
	for _, playlist := range playlists.Playlists {
		if playlist.Name == playlistName {
			playlistID = playlist.ID
			break
		}
	}

	if playlistID == "" {
		return fmt.Errorf("playlist '%s' not found", playlistName)
	}

	// Get playlist tracks
	playlistItems, err := p.Client.GetPlaylistItems(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("error getting playlist items: %v", err)
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
	header := []string{"Track Name", "Artist Name", "Album Name", "Track ID", "Artist ID", "Album ID", "Track URL"}
	writer.Write(header)

	// Write tracks
	for _, item := range playlistItems.Items {
		track := item.Track.Track
		artistNames := ""
		artistIDs := ""

		for i, artist := range track.Artists {
			artistNames += artist.Name
			artistIDs += string(artist.ID)
			if i < len(track.Artists)-1 {
				artistNames += ", "
				artistIDs += ", "
			}
		}

		trackURL := fmt.Sprintf("https://open.spotify.com/track/%s", track.ID)
		row := []string{
			track.Name,
			artistNames,
			track.Album.Name,
			string(track.ID),
			artistIDs,
			string(track.Album.ID),
			trackURL,
		}

		writer.Write(row)
	}

	fmt.Printf("Playlist exported to %s successfully\n", fileName)
	return nil
}

func (p *SpotifyPorter) Import() error {
	if p.Client == nil {
		return fmt.Errorf("not authenticated, call Auth() first")
	}

	ctx := context.Background()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter CSV file path to import: ")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

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

	// Find the track ID column index
	trackIDIndex := -1
	header := records[0]
	for i, colName := range header {
		if strings.Contains(strings.ToLower(colName), "track id") {
			trackIDIndex = i
			break
		}
	}

	if trackIDIndex == -1 {
		return fmt.Errorf("track ID column not found in CSV")
	}

	// Extract track IDs
	var trackIDs []spotify.ID
	var skippedTracks int

	for _, record := range records[1:] {
		if len(record) <= trackIDIndex || record[trackIDIndex] == "" {
			skippedTracks++
			continue
		}

		trackID := spotify.ID(record[trackIDIndex])
		trackIDs = append(trackIDs, trackID)

		// Spotify API has a limit of 100 tracks per request
		if len(trackIDs) >= 100 {
			break
		}
	}

	if len(trackIDs) == 0 {
		return fmt.Errorf("no valid track IDs found in CSV")
	}

	// Get current user
	user, err := p.Client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("error getting current user: %v", err)
	}

	// Create a new playlist
	playlistDescription := fmt.Sprintf("Imported playlist using Soundporter on %s", time.Now().Format("2006-01-02"))
	playlist, err := p.Client.CreatePlaylistForUser(ctx, user.ID, playlistName, playlistDescription, false, false)
	if err != nil {
		return fmt.Errorf("error creating playlist: %v", err)
	}

	// Add tracks to the playlist
	_, err = p.Client.AddTracksToPlaylist(ctx, playlist.ID, trackIDs...)
	if err != nil {
		return fmt.Errorf("error adding tracks to playlist: %v", err)
	}

	fmt.Printf("Successfully created playlist '%s' with %d tracks\n", playlistName, len(trackIDs))
	if skippedTracks > 0 {
		fmt.Printf("Skipped %d tracks due to missing or invalid track IDs\n", skippedTracks)
	}

	return nil
}

// completeAuth is the callback handler for the Spotify auth flow
// It exchanges the authorization code for an access token and writes it to the channel
func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}

func generateState() string {
	bytes := make([]byte, 8) // 8 bytes will result in 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}
