package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dgrinbergs/universal-music-library/music"
	"github.com/spf13/cobra"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export <provider>",
	Short: "Export library from a provider to a local file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := music.Get(args[0])
		if err != nil {
			return err
		}

		ctx := cmd.Context()

		playlists, err := provider.GetPlaylists(ctx)
		if err != nil {
			return fmt.Errorf("fetching playlists: %w", err)
		}

		savedTracks, err := provider.GetSavedTracks(ctx)
		if err != nil {
			return fmt.Errorf("fetching saved tracks: %w", err)
		}

		library := music.Library{
			SavedTracks: savedTracks,
			Playlists:   playlists,
		}

		data, err := json.MarshalIndent(library, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling library: %w", err)
		}

		if exportOutput == "" {
			exportOutput = fmt.Sprintf("%s-library.json", provider.Name())
		}

		if err := os.WriteFile(exportOutput, data, 0o644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		fmt.Printf("Exported %d playlists and %d saved tracks to %s\n",
			len(playlists), len(savedTracks), exportOutput)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path (default: <provider>-library.json)")
	rootCmd.AddCommand(exportCmd)
}
