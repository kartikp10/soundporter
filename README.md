# Soundporter

Soundporter is a command-line interface (CLI) tool designed to help users export and import playlists from and to various music platforms. This tool simplifies the process of managing your music playlists across different services.

## Features

- Export playlists from a specified music platform to various formats.
- Import playlists from different formats into your preferred music platform.
- Utility functions for input validation and playlist formatting.

## Installation

To install Soundporter, clone the repository and navigate to the project directory:

```bash
git clone <repository-url>
cd soundporter
```

Then, build the project using Go:

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
  - Example: `./soundporter export --source <platform> --destination <format>`

- **import**: Import playlists into a music platform.
  - Example: `./soundporter import --source <format> --destination <platform>`

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any enhancements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.