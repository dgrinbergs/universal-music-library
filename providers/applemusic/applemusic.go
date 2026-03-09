package applemusic

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
	return "apple-music"
}

func (p *Provider) Authenticate(ctx context.Context) error {
	return authenticate(ctx)
}

func (p *Provider) GetPlaylists(ctx context.Context) ([]music.Playlist, error) {
	amPlaylists, err := getAllPlaylists()
	if err != nil {
		return nil, err
	}

	// Note: Apple Music library playlists don't include track details inline.
	// A full implementation would fetch tracks for each playlist.
	playlists := make([]music.Playlist, 0, len(amPlaylists))
	for _, ap := range amPlaylists {
		desc := ""
		if ap.Attributes.Description != nil {
			desc = ap.Attributes.Description.Standard
		}
		playlists = append(playlists, music.Playlist{
			Name:        ap.Attributes.Name,
			Description: desc,
		})
	}
	return playlists, nil
}

func (p *Provider) GetSavedTracks(ctx context.Context) ([]music.Track, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (p *Provider) SearchTrack(ctx context.Context, track music.Track) (*music.Track, error) {
	storefront := getStorefront()
	found, err := findTrack(storefront, track)
	if err != nil {
		return nil, err
	}
	result := found.toTrack()
	return &result, nil
}

func (p *Provider) CreatePlaylist(ctx context.Context, name string, tracks []music.Track) error {
	storefront := getStorefront()

	playlist, err := createPlaylist(name)
	if err != nil {
		return fmt.Errorf("creating playlist: %w", err)
	}

	songIDs := make([]string, 0, len(tracks))
	for i, t := range tracks {
		found, err := findTrack(storefront, t)
		if err != nil {
			slog.Warn("Track not found on Apple Music", "track", t.Title, "artist", t.Artist)
			continue
		}
		songIDs = append(songIDs, found.ID)

		if (i+1)%50 == 0 {
			fmt.Printf("  Matched %d/%d tracks...\n", i+1, len(tracks))
		}
	}

	if len(songIDs) > 0 {
		if err := addTracksToPlaylist(playlist.ID, songIDs); err != nil {
			return fmt.Errorf("adding tracks to playlist: %w", err)
		}
	}

	slog.Info("Created playlist on Apple Music",
		"name", name, "matched", len(songIDs), "total", len(tracks))
	return nil
}

func (p *Provider) SaveTracks(ctx context.Context, tracks []music.Track) error {
	storefront := getStorefront()

	songIDs := make([]string, 0, len(tracks))
	for i, t := range tracks {
		found, err := findTrack(storefront, t)
		if err != nil {
			slog.Warn("Track not found on Apple Music", "track", t.Title, "artist", t.Artist)
			continue
		}
		songIDs = append(songIDs, found.ID)

		if (i+1)%50 == 0 {
			fmt.Printf("  Matched %d/%d tracks...\n", i+1, len(tracks))
		}
	}

	if len(songIDs) > 0 {
		if err := addSongsToLibrary(songIDs); err != nil {
			return fmt.Errorf("adding songs to library: %w", err)
		}
	}

	slog.Info("Saved tracks to Apple Music library",
		"matched", len(songIDs), "total", len(tracks))
	return nil
}
