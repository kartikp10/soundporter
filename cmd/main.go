package main

import (
	"fmt"
	"os"
	"soundporter/internal/actions"
	"strings"

	"github.com/charmbracelet/huh"
	_ "github.com/joho/godotenv/autoload"
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "from",
						Usage:    "Platform to export from (spotify, youtube)",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "file",
						Usage:    "File path to save the exported playlists (default: playlists.csv)",
						Required: false,
					},
				},
				Action: actions.ExportPlaylist,
			},
			{
				Name:  "import",
				Usage: "Import playlists to a platform",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "to",
						Aliases:  []string{"t"},
						Usage:    "Platform to import to (spotify, youtube)",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "CSV file to import",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					platform := strings.ToLower(c.String("to"))
					sourceFile := c.String("file")
					if platform == "" {
						huh.NewSelect[string]().
							Title("Choose the platform to import to").
							Options(
								huh.NewOption("Spotify", "spotify"),
								huh.NewOption("YouTube Music", "youtube"),
							).
							Value(&platform).
							Run()
					}
					fmt.Println("Not implemented yet. Will importing from file:", sourceFile)
					return nil
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
