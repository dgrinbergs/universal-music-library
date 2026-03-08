package cmd

import (
	"fmt"

	"github.com/dgrinbergs/universal-music-library/music"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources from a provider",
}

var listPlaylistsCmd = &cobra.Command{
	Use:   "playlists <provider>",
	Short: "List playlists from a provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := music.Get(args[0])
		if err != nil {
			return err
		}

		playlists, err := provider.GetPlaylists(cmd.Context())
		if err != nil {
			return err
		}

		for _, p := range playlists {
			fmt.Printf("%-40s (%d tracks)\n", p.Name, len(p.Tracks))
		}
		return nil
	},
}

var listTracksCmd = &cobra.Command{
	Use:   "tracks <provider>",
	Short: "List saved tracks from a provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := music.Get(args[0])
		if err != nil {
			return err
		}

		tracks, err := provider.GetSavedTracks(cmd.Context())
		if err != nil {
			return err
		}

		for _, t := range tracks {
			fmt.Printf("%s — %s\n", t.Artist, t.Title)
		}
		return nil
	},
}

func init() {
	listCmd.AddCommand(listPlaylistsCmd)
	listCmd.AddCommand(listTracksCmd)
	rootCmd.AddCommand(listCmd)
}
