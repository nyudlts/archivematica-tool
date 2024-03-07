package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	amatica "github.com/nyudlts/go-archivematica"
	"github.com/spf13/cobra"
)

var (
	directoryName string
	writer        *bufio.Writer
	windows       bool
)

const locationName = "amatica rws ingest point"

func init() {
	transferCmd.Flags().StringVar(&config, "config", "", "")
	transferCmd.Flags().StringVar(&directoryName, "path", "", "")
	transferCmd.Flags().BoolVar(&windows, "windows", false, "")
	rootCmd.AddCommand(transferCmd)
}

var transferCmd = &cobra.Command{
	Use: "transfer",
	Run: func(cmd *cobra.Command, args []string) {

		//create a client
		var err error
		client, err = amatica.NewAMClient(config, 20)
		if err != nil {
			panic(err)
		}

		//create an output file
		of, err := os.Create("amatica-transfers.txt")
		if err != nil {
			panic(err)
		}
		defer of.Close()
		writer = bufio.NewWriter(of)

		//process the directory
		xfrDirs, err := os.ReadDir(directoryName)
		if err != nil {
			panic(err)
		}

		for _, xferDir := range xfrDirs {
			if strings.Contains(xferDir.Name(), "fales_") || strings.Contains(xferDir.Name(), "tamwag_") {
				xferPath := filepath.Join(directoryName, xferDir.Name())
				xipPath := strings.ReplaceAll(xferPath, client.StagingLoc, "")

				if err := transferPackage(xipPath); err != nil {
					panic(err)
				}

			}
		}

	},
}

func transferPackage(xipPath string) error {

	//get the transfer directory location
	location, err := client.GetLocationByName(locationName)
	if err != nil {
		return err
	}

	//construct the filepath
	amXIPPath := filepath.Join(location.Path, xipPath)
	if windows {
		//convert the path seprators if on windows
		amXIPPath = strings.Replace(amXIPPath, "\\", "/", -1)
	}

	//request to transfer the xip
	startTransferResponse, err := client.StartTransfer(location.UUID, amXIPPath)
	if err != nil {
		return err
	}

	//catch the soft error
	if regexp.MustCompile("^Error").MatchString(startTransferResponse.Message) {
		return fmt.Errorf("%s", startTransferResponse.Message)
	}

	fmt.Println(startTransferResponse.Message)

	//get the uuid for the transfer
	uuid, err := startTransferResponse.GetUUID()
	if err != nil {
		panic(err)
	}

	fmt.Printf("transfer started: %v %s", uuid, filepath.Base(amXIPPath))

	foundUnapproved := false

	for !foundUnapproved {
		foundUnapproved = findUnapprovedTransfer(uuid)
		if !foundUnapproved {
			fmt.Println("  * Waiting for approval process to complete")
			time.Sleep(5 * time.Second)
		}
	}

	//approve the transfer

	transfer, err := client.GetTransferStatus(uuid)
	if err != nil {
		return err
	}

	if err := client.ApproveTransfer(transfer.Directory, "standard"); err != nil {
		return err
	}

	approvedTransfer, err := client.GetTransferStatus(uuid)
	if err != nil {
		return err
	}

	fmt.Println("Transfer approved:", approvedTransfer.UUID)

	//change this logic over to a channel
	foundCompleted := false
	for !foundCompleted {
		foundCompleted = findCompletedTransfer(uuid)
		if !foundCompleted {
			fmt.Println("  * Waiting for transfer process to complete")
			time.Sleep(5 * time.Second)
		}
	}

	completedTransfer, err := client.GetTransferStatus(uuid)
	if err != nil {
		return err
	}

	sipUUID := completedTransfer.SIPUUID

	fmt.Println("Transfer completed, SIPUUID:", sipUUID)

	//change this logic over to a channel
	foundIngestCompleted := false
	for !foundIngestCompleted {
		foundIngestCompleted = findCompletedIngest(sipUUID)
		if !foundIngestCompleted {
			fmt.Println("  * Waiting for ingest process to complete")
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("Ingest Completed:", sipUUID)
	aipDir, err := client.GetAIPLocation(sipUUID)
	if err != nil {
		return err
	}
	fmt.Println(aipDir)
	writer.WriteString(fmt.Sprintf("%s\n", aipDir))
	writer.Flush()

	return nil

}

func findCompletedIngest(sipUuid string) bool {
	completedIngests, err := client.GetCompletedIngests()
	if err != nil {
		panic(err)
	}

	completedIngestsMap, err := client.GetCompletedIngestsMap(completedIngests)
	if err != nil {
		panic(err)
	}

	for k, _ := range completedIngestsMap {
		if k == sipUuid {
			return true
		}
	}

	return false
}

func findCompletedTransfer(uuid string) bool {
	completedTransfers, err := client.GetCompletedTransfers()
	if err != nil {
		panic(err)
	}

	completedTransfersMap, err := client.GetCompletedTransfersMap(completedTransfers)
	if err != nil {
		panic(err)

	}

	for k, _ := range completedTransfersMap {
		if k == uuid {
			return true
		}
	}

	return false
}

func findUnapprovedTransfer(uuid string) bool {
	unapprovedTransfers, err := client.GetUnapprovedTransfers()
	if err != nil {
		panic(err)

	}

	unapprovedTransfersMap, err := client.GetUnapprovedTransfersMap(unapprovedTransfers)
	if err != nil {
		panic(err)
	}

	//find the unapproved transfer
	for k, _ := range unapprovedTransfersMap {
		if k == uuid {
			return true
		}
	}

	return false
}
