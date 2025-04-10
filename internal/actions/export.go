package actions

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"soundporter/internal/playlist"
	"soundporter/internal/porter"
	"soundporter/internal/utils"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/urfave/cli/v2"
)

func ExportPlaylist(c *cli.Context) error {
	platform := strings.ToLower(c.String("from"))
	destFile := c.String("file")

	if platform == "" {
		huh.NewSelect[string]().
			Title("Choose the platform to export from").
			Options(
				huh.NewOption("Spotify", "spotify"),
				huh.NewOption("YouTube Music", "youtube"),
			).
			Value(&platform).
			Run()
	}
	// initialize porter
	p, err := porter.NewPorterWithCredentials(platform, "", "")
	if err != nil {
		return fmt.Errorf("failed to create porter for platform %s: %v", platform, err)
	}

	// handle auth
	p.Authenticate()

	var playlistId string
	huh.NewInput().
		Title("Enter the file path to save the exported playlists").
		Value(&destFile).
		Run()
	huh.NewSelect[string]().
		Height(10).
		Title("Choose a playlist to export").
		OptionsFunc(func() []huh.Option[string] {
			playlists, err := p.GetPlaylists()
			if err != nil {
				return nil
			}
			return getPlaylistOptions(playlists)
		}, &p).
		Value(&playlistId).
		Run()

	ctx := context.Background()
	download := func(ctx context.Context) error {
		// get tracks
		tracks, err := p.GetPlaylistTracks(playlistId)
		if err != nil {
			return fmt.Errorf("failed to get tracks for playlist %s: %v", playlistId, err)
		}

		// Open the CSV file for writing
		file, err := os.Create(destFile)
		if err != nil {
			return err
		}
		defer file.Close()

		headers := utils.StructToCsvHeader(reflect.TypeOf(playlist.Track{}))
		utils.WriteToCsvFile(destFile, headers, tracks)
		return nil
	}

	err = spinner.New().Title("Exporting...").Context(ctx).ActionWithErr(download).Run()

	return err
}

func getPlaylistOptions(p []playlist.Playlist) []huh.Option[string] {
	playlistOptions := make([]huh.Option[string], len(p))
	for i, pl := range p {
		playlistOptions[i] = huh.NewOption(pl.Name, pl.ID)
	}
	return playlistOptions
}
