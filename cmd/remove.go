package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	amatica "github.com/nyudlts/go-archivematica"
	"github.com/spf13/cobra"
)

var (
	packageType   string
	packageStatus string
	packagePath   string
	typePtr       *string
	statusPtr     *string
	pathPtr       *string
	test          bool
)

func init() {
	removeCmd.Flags().StringVar(&config, "config", "", "")
	removeCmd.Flags().StringVar(&packageType, "type", "", "")
	removeCmd.Flags().StringVar(&packageStatus, "status", "", "")
	removeCmd.Flags().StringVar(&packagePath, "path", "", "")
	removeCmd.Flags().BoolVar(&test, "test", false, "")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use: "remove",
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

		for _, pack := range packs {
			pathSplit := strings.Split(pack.CurrentPath, "/")
			fmt.Println("requesting deletion of:", pathSplit[len(pathSplit)-1])
			if err := requestDeletion(pack.UUID); err != nil {
				panic(err)
			}
		}
	},
}

func setPointers() {
	if packageType == "" {
		typePtr = nil
	} else {
		typePtr = &packageType
	}

	if packageStatus == "" {
		statusPtr = nil
	} else {
		statusPtr = &packageStatus
	}

	if packagePath == "" {
		pathPtr = nil
	} else {
		pathPtr = &packagePath
	}
}

func requestDeletion(packageUUID uuid.UUID) error {

	fmt.Printf("Requesting deletion for `%s`\n", packageUUID)

	pack, err := client.GetPackage(packageUUID)
	if err != nil {
		return err
	}

	dr := amatica.DeletionRequest{}
	dr.EventReason = "Transferred to R*"
	dr.UserEmail = client.SSUserEmail
	dr.UserID = client.SSUserID
	pipelineUUID, err := pack.GetPipelineUUID()
	if err != nil {
		return err
	}
	dr.Pipeline = pipelineUUID

	drBytes, err := json.Marshal(dr)
	if err != nil {
		return err
	}

	if !test {
		msg, err := client.RequestPackageDeletion(pack.UUID, string(drBytes))
		if err != nil {
			return err
		}

		fmt.Println(msg)
	} else {
		fmt.Println("Test mode, not requesting deletion")
		fmt.Println(dr)
	}
	return nil
}
