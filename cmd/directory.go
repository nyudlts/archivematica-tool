package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nyudlts/bytemath"
	"github.com/spf13/cobra"
)

type ProblemER struct {
	Dir   string
	Count int
	Size  int64
}

var (
	rootDir    string
	ProblemERs = []ProblemER{}
)

func init() {
	directoryCmd.Flags().StringVar(&rootDir, "root-dir", "", "")
	rootCmd.AddCommand(directoryCmd)
}

var directoryCmd = &cobra.Command{
	Use: "check-dir",
	Run: func(cmd *cobra.Command, args []string) {
		fi, err := os.Stat(rootDir)
		if err != nil {
			panic(err)
		}

		if !fi.IsDir() {
			panic(fmt.Errorf("%s is not a directory", rootDir))
		}

		entries, err := os.ReadDir(rootDir)
		if err != nil {
			panic(err)
		}

		var totalCount int = 0
		var totalSize int64 = 0

		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Print("scanning ", entry.Name(), ": ")
				count, size, err := countFiles(filepath.Join(rootDir, entry.Name()))
				if err != nil {
					panic(err)
				}

				totalCount = totalCount + count
				totalSize = totalSize + size

				if count >= 2000 || size > 214748364800 {
					ProblemERs = append(ProblemERs, ProblemER{entry.Name(), count, size})
				}

				fmt.Println("count:", count, "size:", bytemath.ConvertBytesToHumanReadable(size))
			}
		}

		fmt.Println("Total Count:", totalCount)
		fmt.Println("Total Size:", bytemath.ConvertBytesToHumanReadable(totalSize))

		if len(ProblemERs) > 0 {
			fmt.Println("Problem ERs")
			for _, er := range ProblemERs {
				fmt.Println(er)
			}
		} else {
			fmt.Println("No Problem ERs Found")
		}

	},
}

func countFiles(dir string) (int, int64, error) {
	count := 0
	var size int64 = 0

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			count = count + 1
			size = size + info.Size()

		}
		return nil
	})

	if err != nil {
		return count, size, err
	}

	return count, size, nil

}
