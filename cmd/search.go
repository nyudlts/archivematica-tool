package cmd

import (
	"encoding/json"
	"fmt"

	amatica "github.com/nyudlts/go-archivematica"
	"github.com/spf13/cobra"
)

func init() {
	searchCmd.Flags().StringVar(&config, "config", "", "")
	searchCmd.Flags().StringVar(&packageType, "type", "", "")
	searchCmd.Flags().StringVar(&packageStatus, "status", "", "")
	searchCmd.Flags().StringVar(&packagePath, "path", "", "")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use: "search",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		client, err = amatica.NewAMClient(config, 20)
		if err != nil {
			panic(err)
		}

		setPointers()

		packs, err := client.FilterPackages(typePtr, statusPtr, pathPtr)
		if err != nil {
			panic(err)
		}

		for i, pack := range packs {
			b, err := json.Marshal(pack)
			if err != nil {
				panic(err)
			}

			fmt.Printf("%d.  %s\n\n", i+1, string(b))
		}
	},
}
