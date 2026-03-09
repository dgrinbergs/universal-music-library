package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/dgrinbergs/universal-music-library/music"
	"github.com/spf13/cobra"
)

var (
	importPlaylists   bool
	importSavedTracks bool
)

var importCmd = &cobra.Command{
	Use:   "import <provider> <file>",
	Short: "Import library from a JSON file into a provider",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := music.Get(args[0])
		if err != nil {
			return err
		}

		data, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		var library music.Library
		if err := json.Unmarshal(data, &library); err != nil {
			return fmt.Errorf("parsing library file: %w", err)
		}

		ctx := cmd.Context()

		if importPlaylists && len(library.Playlists) > 0 {
			fmt.Printf("Importing %d playlists into %s...\n", len(library.Playlists), provider.Name())
			for _, p := range library.Playlists {
				fmt.Printf("  Importing playlist: %s (%d tracks)\n", p.Name, len(p.Tracks))
				if err := provider.CreatePlaylist(ctx, p.Name, p.Tracks); err != nil {
					slog.Warn("Failed to import playlist", "name", p.Name, "err", err)
					continue
				}
			}
		}

		if importSavedTracks && len(library.SavedTracks) > 0 {
			fmt.Printf("Importing %d saved tracks into %s...\n", len(library.SavedTracks), provider.Name())
			if err := provider.SaveTracks(ctx, library.SavedTracks); err != nil {
				return fmt.Errorf("importing saved tracks: %w", err)
			}
		}

		fmt.Println("Import complete.")
		return nil
	},
}

func init() {
	importCmd.Flags().BoolVar(&importPlaylists, "playlists", true, "import playlists")
	importCmd.Flags().BoolVar(&importSavedTracks, "saved-tracks", true, "import saved tracks")
	rootCmd.AddCommand(importCmd)
}
