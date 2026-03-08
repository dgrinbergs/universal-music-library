package cmd

import (
	"github.com/spf13/cobra"

	_ "github.com/dgrinbergs/universal-music-library/config"
	_ "github.com/dgrinbergs/universal-music-library/providers/spotify"
)

var rootCmd = &cobra.Command{
	Use:   "uml",
	Short: "Universal Music Library — sync playlists across streaming platforms",
}

func Execute() error {
	return rootCmd.Execute()
}
