package cmd

import (
	amatica "github.com/nyudlts/go-archivematica"
	"github.com/spf13/cobra"
)

const version = "1.0.0"

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
