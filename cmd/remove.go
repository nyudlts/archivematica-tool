package cmd

import "github.com/spf13/cobra"

func init() {
	removeCmd.Flags().StringVar(&config, "config", "", "")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use: "remove-transferred",
}
