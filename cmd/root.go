package cmd

import (
	amatica "github.com/nyudlts/go-archivematica"
	"github.com/spf13/cobra"
)

var (
	config string
	client *amatica.AMClient
)

var rootCmd = &cobra.Command{
	Use: "",
	Run: func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	rootCmd.Execute()
}
