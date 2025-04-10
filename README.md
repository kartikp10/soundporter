![logo](./banner.png)

# Soundporter

Soundporter is a CLI tool that exports and imports playlists from and to various music platforms. This is a hobby project primarily to help me move all my playlists from Spotify to YouTube Music and (maybe) Apple Music too. Feel free to use, modify and contribute!

## Features

- Export playlists from a specified music platform to various formats.
- Import playlists from different formats into your preferred music platform.

## Installation

To install Soundporter, clone the repository and build the project using Go:

```bash
go build -o soundporter ./cmd
```

## Usage

To use Soundporter, run the following command in your terminal:

```bash
./soundporter [command]
```

### Commands

- **export**: Export playlists from a music platform.
  - Example: `./soundporter export`

- **import**: Import playlists into a music platform.
  - Example: `./soundporter import`

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any enhancements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.
