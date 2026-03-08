package cmd

import (
	"fmt"

	"github.com/dgrinbergs/universal-music-library/music"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth <provider>",
	Short: "Authenticate with a music provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := music.Get(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Authenticating with %s...\n", provider.Name())
		return provider.Authenticate(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
