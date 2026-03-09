# Universal Music Library

Sync playlists and saved tracks across streaming platforms.

Universal Music Library (UML) is a command-line tool that exports, imports, and syncs your music library between different streaming services. It uses ISRC codes for accurate track matching with a title+artist fallback.

## Supported Services

- **Spotify** — full support (playlists, saved tracks, search)
- **Apple Music** — partial support (playlists, saved tracks, search)

## Installation

```bash
go build -o uml .
```

## Configuration

UML stores its configuration at `~/.config/universal-music-library/config.yaml`.

### Spotify

1. Create an app at https://developer.spotify.com/dashboard
2. Set the redirect URI to `http://127.0.0.1:8080/callback`
3. Add your client ID to the config:

```yaml
spotify:
  client_id: YOUR_CLIENT_ID
```

### Apple Music

1. Enable MusicKit in your Apple Developer account
2. Create a MusicKit private key (`.p8` file)
3. Add your credentials to the config:

```yaml
apple_music:
  team_id: YOUR_TEAM_ID
  key_id: YOUR_KEY_ID
  private_key_path: /path/to/AuthKey_XXXX.p8
  storefront: us
```

## Usage

### Authenticate

```bash
uml auth spotify
uml auth apple-music
```

### Export

```bash
uml export spotify                     # writes to spotify-library.json
uml export spotify -o backup.json      # custom output file
```

### Import

```bash
uml import apple-music spotify-library.json                # import everything
uml import apple-music data.json --playlists=true          # playlists only
uml import apple-music data.json --saved-tracks=true       # saved tracks only
```

### Sync

```bash
uml sync spotify apple-music                               # sync everything
uml sync spotify apple-music --playlists=true              # playlists only
uml sync spotify apple-music --saved-tracks=true           # saved tracks only
```

### List

```bash
uml list playlists spotify
uml list tracks spotify
```

## How It Works

1. **Export** fetches all playlists and saved tracks from a service and writes them to a JSON file using a universal format
2. **Import** reads a library JSON file, searches for each track on the destination service, and creates playlists / saves tracks
3. **Sync** combines both — reads from the source service and writes directly to the destination

Track matching prioritizes ISRC (International Standard Recording Code) for exact matches, falling back to title + artist search.

## Project Structure

```
cmd/           CLI commands (auth, export, import, sync, list)
music/         Core models (Track, Playlist, Library) and provider interface
providers/     Service implementations (spotify, applemusic)
config/        Configuration management
```

## License

MIT
