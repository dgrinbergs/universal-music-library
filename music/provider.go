package music

import (
	"context"
	"fmt"
)

type Provider interface {
	Name() string
	Authenticate(ctx context.Context) error
	GetPlaylists(ctx context.Context) ([]Playlist, error)
	GetSavedTracks(ctx context.Context) ([]Track, error)
	SearchTrack(ctx context.Context, track Track) (*Track, error)
	CreatePlaylist(ctx context.Context, name string, tracks []Track) error
}

var providers = map[string]Provider{}

func Register(p Provider) {
	providers[p.Name()] = p
}

func Get(name string) (Provider, error) {
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return p, nil
}

func ListProviders() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
