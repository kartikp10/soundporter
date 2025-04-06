package main

import (
	"fmt"
	"os"
	porter "soundporter/internal/porter"
	spotifyPorter "soundporter/internal/spotify"
	youtubePorter "soundporter/internal/youtube"
	"strings"

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
						Aliases:  []string{"f"},
						Usage:    "Platform to export from (spotify, youtube)",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					platform := strings.ToLower(c.String("from"))
					var port *porter.Port

					switch platform {
					case "spotify":
						port = porter.NewPort(platform, &spotifyPorter.SpotifyPorter{})
					case "youtube":
						port = porter.NewPort(platform, &youtubePorter.YouTubePorter{})
					default:
						return fmt.Errorf("unsupported platform: %s", platform)
					}

					err := port.Auth()
					if err != nil {
						return fmt.Errorf("authentication failed: %v", err)
					}
					return port.Export()
				},
			},
			{
				Name:  "import",
				Usage: "Import playlists to a platform",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "to",
						Aliases:  []string{"t"},
						Usage:    "Platform to import to (spotify, youtube)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "CSV file to import",
						Required: false,
					},
				},
				Action: func(c *cli.Context) error {
					platform := strings.ToLower(c.String("from"))
					var port *porter.Port

					switch platform {
					case "spotify":
						port = porter.NewPort(platform, &spotifyPorter.SpotifyPorter{})
					case "youtube":
						port = porter.NewPort(platform, &youtubePorter.YouTubePorter{})
					default:
						return fmt.Errorf("unsupported platform: %s", platform)
					}

					err := port.Auth()
					if err != nil {
						return fmt.Errorf("authentication failed: %v", err)
					}
					return port.Import()
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
