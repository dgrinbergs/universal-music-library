package spotify

import (
	"context"
	"fmt"

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
	// TODO: implement OAuth2 PKCE flow
	// 1. Start local HTTP server for callback
	// 2. Open browser to Spotify auth URL
	// 3. Exchange code for tokens
	// 4. Store refresh token in config
	return fmt.Errorf("spotify auth not yet implemented")
}

func (p *Provider) GetPlaylists(ctx context.Context) ([]music.Playlist, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (p *Provider) GetSavedTracks(ctx context.Context) ([]music.Track, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (p *Provider) SearchTrack(ctx context.Context, track music.Track) (*music.Track, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (p *Provider) CreatePlaylist(ctx context.Context, name string, tracks []music.Track) error {
	return fmt.Errorf("not yet implemented")
}
