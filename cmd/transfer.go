package cmd

import (
	"bufio"
	"fmt"
	"log"
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

		fmt.Println("Creating Log File")
		logFile, err := os.Create("am-tools-transfer.log")
		if err != nil {
			panic(err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)

		//set the poll time
		fmt.Printf("setting polling time to %d seconds\n", delayTime)
		log.Printf("INFO setting polling time to %d seconds", delayTime)
		delay = time.Duration(delayTime)

		//create a client
		fmt.Println("creating go-archivematica client")
		log.Println("INFO creating go-archivematica client")
		client, err = amatica.NewAMClient(config, 20)
		if err != nil {
			panic(err)
		}

		//create an output file
		fmt.Println("creating aip-file.txt")
		log.Println("INFO creating aip-file.txt")
		of, err := os.Create("aip-file.txt")
		if err != nil {
			panic(err)
		}
		defer of.Close()
		writer = bufio.NewWriter(of)

		//process the directory
		fmt.Printf("Reading source directory: %s", directoryName)
		log.Printf("INFO reading source directory: %s", directoryName)

		xfrDirs, err := os.ReadDir(directoryName)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Transferring files from %s", directoryName)
		log.Printf("INFO transferring files from %s", directoryName)

		for _, xferDir := range xfrDirs {
			if strings.Contains(xferDir.Name(), "fales_") || strings.Contains(xferDir.Name(), "tamwag_") {
				xferPath := filepath.Join(directoryName, xferDir.Name())
				xipPath := strings.ReplaceAll(xferPath, client.StagingLoc, "")

				if err := transferPackage(xipPath); err != nil {
					fmt.Printf("ERROR %s", strings.ReplaceAll(err.Error(), "\n", ""))
					log.Printf("ERROR %s", strings.ReplaceAll(err.Error(), "\n", ""))
				}

			}
		}
	},
}

func transferPackage(xipPath string) error {
	fmt.Printf("Transfering package: %s\n", filepath.Base(xipPath))
	log.Printf("INFO transfering package: %s", filepath.Base(xipPath))
	//get the transfer directory location
	location, err := client.GetLocationByName(locationName)
	if err != nil {
		return err
	}

	//construct the filepath
	amXIPPath := filepath.Join(location.Path, xipPath)
	fmt.Printf("Creating path of SIP: %s\n", amXIPPath)
	log.Printf("INFO creating path of SIP: %s", amXIPPath)

	//convert the path seprators if on windows
	if windows {
		fmt.Println("INFO converting windows path seperators to linux")
		log.Println("INFO converting windows path seperators to linux")
		amXIPPath = strings.Replace(amXIPPath, "\\", "/", -1)
	}
	fmt.Printf("INFO SIP Path: %s\n", amXIPPath)
	log.Printf("INFO SIP path: %s", amXIPPath)

	//request to transfer the xip
	fmt.Printf("Requesting Transfer for %s\n", amXIPPath)
	log.Printf("INFO requesting Transfer for %s\n", amXIPPath)

	startTransferResponse, err := client.StartTransfer(location.UUID, amXIPPath)
	if err != nil {
		return err
	}

	//catch the soft error
	if regexp.MustCompile("^Error").MatchString(startTransferResponse.Message) {
		return fmt.Errorf("%s", startTransferResponse.Message)
	}

	fmt.Printf("\nStart Transfer Request Message: %s\n", startTransferResponse.Message)
	log.Printf("INFO start Transfer Request Message: %s", startTransferResponse.Message)

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
	fmt.Printf("Approving Transfer %s\n", uuid)
	log.Printf("INFO approving Transfer %s", uuid)
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

	fmt.Printf("transfer processing started: %s\n", filepath.Base(amXIPPath))
	log.Printf("INFO transfer processing started: %s", filepath.Base(amXIPPath))
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

		if ts.Status == "" {
			return fmt.Errorf("no status being returned")
		}

		if ts.Status == "COMPLETE" {
			foundCompleted = true
		}

		if !foundCompleted {
			fmt.Printf("  * Transfer Status: %s  Microservice: %s\n", ts.Status, ts.Microservice)
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
	fmt.Printf("Transfer processing completed, SIPUUID: %s\n", sipUUID)
	log.Printf("INFO transfer processing completed, SIPUUID: %s", sipUUID)

	//start Ingest
	fmt.Printf("\nIngest processing started: %s-%s\n", filepath.Base(amXIPPath), sipUUID)
	log.Printf("INFO ingest processing started: %s-%s\n", filepath.Base(amXIPPath), sipUUID)
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
			fmt.Printf("  * Ingest Status: %s  Microservice: %s\n", is.Status, is.Microservice)
			time.Sleep(delay * time.Second)
		}
	}

	fmt.Printf("Ingest Completed: %s", sipUUID)
	log.Printf("INFO ingest Completed: %s", sipUUID)
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
	log.Printf("INFO writing path to aip-file: %s", aipPath)
	fmt.Printf("INFO writing path to aip-file: %s\n", aipPath)
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
