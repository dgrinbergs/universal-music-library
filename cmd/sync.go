package cmd

import (
	"fmt"
	"log/slog"

	"github.com/dgrinbergs/universal-music-library/music"
	"github.com/spf13/cobra"
)

var syncPlaylists bool
var syncSavedTracks bool

var syncCmd = &cobra.Command{
	Use:   "sync <source> <destination>",
	Short: "Sync library from one provider to another",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src, err := music.Get(args[0])
		if err != nil {
			return fmt.Errorf("source: %w", err)
		}

		dst, err := music.Get(args[1])
		if err != nil {
			return fmt.Errorf("destination: %w", err)
		}

		ctx := cmd.Context()

		if syncPlaylists {
			slog.Info("Syncing playlists", "from", src.Name(), "to", dst.Name())

			playlists, err := src.GetPlaylists(ctx)
			if err != nil {
				return fmt.Errorf("fetching playlists from %s: %w", src.Name(), err)
			}

			for _, p := range playlists {
				matched := make([]music.Track, 0, len(p.Tracks))
				for _, t := range p.Tracks {
					found, err := dst.SearchTrack(ctx, t)
					if err != nil {
						slog.Warn("Track not found on destination",
							"track", t.Title, "artist", t.Artist, "err", err)
						continue
					}
					matched = append(matched, *found)
				}

				if err := dst.CreatePlaylist(ctx, p.Name, matched); err != nil {
					return fmt.Errorf("creating playlist %q on %s: %w", p.Name, dst.Name(), err)
				}

				slog.Info("Synced playlist",
					"name", p.Name,
					"matched", len(matched),
					"total", len(p.Tracks))
			}
		}

		if syncSavedTracks {
			slog.Info("Syncing saved tracks", "from", src.Name(), "to", dst.Name())
			// TODO: implement saved tracks sync
			return fmt.Errorf("saved tracks sync not yet implemented")
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncPlaylists, "playlists", true, "sync playlists")
	syncCmd.Flags().BoolVar(&syncSavedTracks, "saved-tracks", true, "sync saved tracks")
	rootCmd.AddCommand(syncCmd)
}
