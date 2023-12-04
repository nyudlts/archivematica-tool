package cmd

import (
	"flag"

	amatica "github.com/nyudlts/go-archivematica"

	"github.com/spf13/cobra"
)

func init() {
	monitorCommand.Flags().StringVar(&config, "config", "", "")
	rootCmd.AddCommand(monitorCommand)
}

var monitorCommand = &cobra.Command{
	Use: "monitor",
	Run: func(cmd *cobra.Command, args []string) {
		flag.Parse()
		client, err := amatica.NewAMClient(config, 20)
		if err != nil {
			panic(err)
		}

		client.Monitor()
	},
}
