package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use: "info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("am-tool v%s\n", version)
		bi, ok := debug.ReadBuildInfo()
		if ok != true {
			panic(fmt.Errorf("unable to read build info"))
		}
		fmt.Println("\nBuild Info\n==========")
		fmt.Println(bi)
	},
}
