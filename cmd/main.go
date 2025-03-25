package main

import (
	"fmt"
	"os"
	spotifyPorter "soundporter/internal/spotify"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "soundporter",
		Usage: "Soundporter is a CLI tool to export and import playlists from and to music platforms.",
		Commands: []*cli.Command{
			{
				Name:  "export",
				Usage: "Export playlists from a platform",
				Action: func(c *cli.Context) error {
					porter := spotifyPorter.SpotifyPorter{}
					porter.Auth()
					return porter.Export()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
