package spotify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dgrinbergs/universal-music-library/music"
)

type Provider struct{}

func init() {
	music.Register(&Provider{})
}

func (p *Provider) Name() string {
	return "spotify"
}

func (p *Provider) Authenticate(ctx context.Context) error {
	return authenticate(ctx)
}

func (p *Provider) GetPlaylists(ctx context.Context) ([]music.Playlist, error) {
	spotifyPlaylists, err := getAllPlaylists()
	if err != nil {
		return nil, err
	}

	playlists := make([]music.Playlist, 0, len(spotifyPlaylists))
	for _, sp := range spotifyPlaylists {
		slog.Info("Fetching playlist tracks", "name", sp.Name, "total", sp.Tracks.Total)

		tracks, err := getPlaylistTracks(sp.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching tracks for %q: %w", sp.Name, err)
		}

		musicTracks := make([]music.Track, 0, len(tracks))
		for _, t := range tracks {
			musicTracks = append(musicTracks, t.toTrack())
		}

		playlists = append(playlists, music.Playlist{
			Name:        sp.Name,
			Description: sp.Description,
			Tracks:      musicTracks,
		})
	}
	return playlists, nil
}

func (p *Provider) GetSavedTracks(ctx context.Context) ([]music.Track, error) {
	spotifyTracks, err := getAllSavedTracks()
	if err != nil {
		return nil, err
	}

	tracks := make([]music.Track, 0, len(spotifyTracks))
	for _, t := range spotifyTracks {
		tracks = append(tracks, t.toTrack())
	}
	return tracks, nil
}

func (p *Provider) SearchTrack(ctx context.Context, track music.Track) (*music.Track, error) {
	found, err := searchTrack(track)
	if err != nil {
		return nil, err
	}
	result := found.toTrack()
	return &result, nil
}

func (p *Provider) CreatePlaylist(ctx context.Context, name string, tracks []music.Track) error {
	user, err := getCurrentUser()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	playlist, err := createPlaylist(user.ID, name)
	if err != nil {
		return fmt.Errorf("creating playlist: %w", err)
	}

	uris := make([]string, 0, len(tracks))
	for _, t := range tracks {
		found, err := searchTrack(t)
		if err != nil {
			slog.Warn("Track not found on Spotify", "track", t.Title, "artist", t.Artist)
			continue
		}
		uris = append(uris, found.URI)
	}

	if len(uris) > 0 {
		if err := addTracksToPlaylist(playlist.ID, uris); err != nil {
			return fmt.Errorf("adding tracks: %w", err)
		}
	}

	slog.Info("Created playlist", "name", name, "tracks", len(uris))
	return nil
}
