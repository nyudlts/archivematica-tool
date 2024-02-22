package cmd

import (
	"fmt"

	amatica "github.com/nyudlts/go-archivematica"
	"github.com/spf13/cobra"
)

func init() {
	removeCmd.Flags().StringVar(&config, "config", "", "")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use: "remove-transferred",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		client, err = amatica.NewAMClient(config, 20)
		if err != nil {
			panic(err)
		}

		dips, err := client.GetPackageType("DIP")
		if err != nil {
			panic(err)
		}

		for i, dip := range dips {
			dJson, err := dip.MarshalPack()
			if err != nil {
				panic(err)
			}
			fmt.Println(i, dJson)
		}
	},
}

/*

	uploaded := []uuid.UUID{}
	packages, err := client.GetPackages(nil)
	if err != nil {
		panic(err)
	}
	complete := false

	for !complete {
		packages, err = client.GetPackages(&packages.Meta.Next)
		if err != nil {
			panic(err)
		}

		for _, pack := range packages.Objects {
			if pack.PackageType == "AIP" && pack.Status == "UPLOADED" {
				uploaded = append(uploaded, pack.UUID)
			}
		}

		if packages.Meta.Next == "" {
			complete = true
		}
	}

	for _, packageUUID := range uploaded {
		fmt.Printf("Requesting deletion for `%s`\n", packageUUID)

		pack, err := client.GetPackage(packageUUID)
		if err != nil {
			panic(err)
		}

		dr := amatica.DeletionRequest{}
		dr.EventReason = "Transferred to R*"
		dr.UserEmail = "don.mennerich@nyu.edu"
		dr.UserID = "1"
		pipelineUUID, err := pack.GetPipelineUUID()
		if err != nil {
			panic(err)
		}
		dr.Pipeline = pipelineUUID

		drBytes, err := json.Marshal(dr)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(drBytes))

		msg, err := client.RequestPackageDeletion(pack.UUID, string(drBytes))
		if err != nil {
			panic(err)
		}

		fmt.Println(msg)

		time.Sleep(500 * time.Millisecond)
	}
*/
