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
	delayTime     int32
	delay         time.Duration
)

const locationName = "amatica rws ingest point"

func init() {
	transferCmd.Flags().StringVar(&config, "config", "", "")
	transferCmd.Flags().StringVar(&directoryName, "path", "", "")
	transferCmd.Flags().BoolVar(&windows, "windows", false, "")
	transferCmd.Flags().Int32Var(&delayTime, "delay", 5, "")
	rootCmd.AddCommand(transferCmd)
}

var transferCmd = &cobra.Command{
	Use: "transfer",
	Run: func(cmd *cobra.Command, args []string) {

		//set the poll time
		delay = time.Duration(delayTime)
		fmt.Println("setting delay to", delayTime, "seconds")

		//create a client
		var err error
		client, err = amatica.NewAMClient(config, 20)
		if err != nil {
			panic(err)
		}

		//create an output file
		fmt.Println("creating aip-file.txt")
		of, err := os.Create("aip-file.txt")
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

		fmt.Println("Transferring files from", directoryName)
		for _, xferDir := range xfrDirs {
			if strings.Contains(xferDir.Name(), "fales_") || strings.Contains(xferDir.Name(), "tamwag_") {
				xferPath := filepath.Join(directoryName, xferDir.Name())
				xipPath := strings.ReplaceAll(xferPath, client.StagingLoc, "")

				if err := transferPackage(xipPath); err != nil {
					fmt.Println(err)
				}

			}
		}

	},
}

func transferPackage(xipPath string) error {
	fmt.Println("Transfering package:", filepath.Base(xipPath))
	//get the transfer directory location
	location, err := client.GetLocationByName(locationName)
	if err != nil {
		return err
	}

	//construct the filepath
	amXIPPath := filepath.Join(location.Path, xipPath)
	//convert the path seprators if on windows
	if windows {
		amXIPPath = strings.Replace(amXIPPath, "\\", "/", -1)
	}

	//request to transfer the xip
	fmt.Println("Requesting Transfer for", amXIPPath)
	startTransferResponse, err := client.StartTransfer(location.UUID, amXIPPath)
	if err != nil {
		return err
	}

	//catch the soft error
	if regexp.MustCompile("^Error").MatchString(startTransferResponse.Message) {
		return fmt.Errorf("%s", startTransferResponse.Message)
	}

	fmt.Println("Start Transfer Request Message: ", startTransferResponse.Message)

	//get the uuid for the transfer
	uuid, err := startTransferResponse.GetUUID()
	if err != nil {
		panic(err)
	}

	//change this logic over to a channel
	foundUnapproved := false
	for !foundUnapproved {
		foundUnapproved = findUnapprovedTransfer(uuid)
		if !foundUnapproved {
			fmt.Println("  * Waiting for approval process to complete")
			time.Sleep(delay * time.Second)
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

	fmt.Printf("transfer processing started: %s\n", filepath.Base(amXIPPath))
	//change this logic over to a channel
	foundCompleted := false
	for !foundCompleted {
		ts, err := client.GetTransferStatus(approvedTransfer.UUID.String())
		if err != nil {
			return err
		}

		if ts.Status == "FAILED" {
			return fmt.Errorf(ts.Microservice)
		}

		if ts.Status == "COMPLETE" {
			foundCompleted = true
		}

		if !foundCompleted {
			fmt.Println("  * Transfer Status:", ts.Status, ",  Microservice:", ts.Microservice)
			time.Sleep(delay * time.Second)
		}
	}

	time.Sleep(5 * time.Second)
	completedTransfer, err := client.GetTransferStatus(uuid)
	if err != nil {
		return err
	}

	sipUUID := completedTransfer.SIPUUID
	if sipUUID == "" {
		return fmt.Errorf("no sipuuid returned")
	}
	fmt.Printf("Transfer completed, SIPUUID: %s\n", sipUUID)

	fmt.Printf("\nIngest processing started: %s-%s\n", filepath.Base(amXIPPath), sipUUID)
	//change this logic over to a channel
	foundIngestCompleted := false
	for !foundIngestCompleted {
		is, err := client.GetIngestStatus(sipUUID)
		if err != nil {
			return err
		}

		if is.Status == "FAILED" {
			return fmt.Errorf(is.Microservice)
		}

		if is.Status == "" {
			return fmt.Errorf("no status being returned")
		}

		if is.Status == "COMPLETE" {
			foundIngestCompleted = true
		}

		if !foundIngestCompleted {
			fmt.Println("  * Ingest Status:", is.Status, ",  Microservice:", is.Microservice)
			time.Sleep(delay * time.Second)
		}
	}

	fmt.Println("Ingest Completed:", sipUUID)
	fmt.Println()

	//write url to aip-file.txt
	aipPath, err := amatica.ConvertUUIDToAMDirectory(sipUUID)
	if err != nil {
		return err
	}

	aipPath = filepath.Join(aipPath, fmt.Sprintf("%s-%s", filepath.Base(xipPath), sipUUID))
	if windows {
		aipPath = strings.Replace(aipPath, "\\", "/", -1)
	}

	aipPath = fmt.Sprintf("%s%s", "/mnt/staging/AIPsStore/", aipPath)
	writer.WriteString(fmt.Sprintf("%s\n", aipPath))
	writer.Flush()

	return nil
}

func findCompletedIngest(sipUuid string) bool {
	fmt.Println("checking", sipUuid)
	completedIngests, err := client.GetCompletedIngests()
	if err != nil {
		panic(err)
	}

	completedIngestsMap, err := client.GetCompletedIngestsMap(completedIngests)
	fmt.Println(completedIngestsMap)
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
