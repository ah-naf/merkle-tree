package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ah-naf/merkle-cli/merkle"
)

func main() {
	inputFile := flag.String("in", "", "Path to file with data blocks")
	dirPath := flag.String("dir", "", "Path to directory to build tree from (mutually exclusive with --in)")
	outputFile := flag.String("out", "output.json", "Path to write Merkle tree JSON")
	delimiter := flag.String("delimiter", "\n", "Delimiter to split records in the input file")

	flag.Parse()

	if (*inputFile == "" && *dirPath == "") || (*inputFile != "" && *dirPath != "") {
		fmt.Fprintln(os.Stderr, "Error: either --in or --dir must be provided, but not both")
		flag.Usage()
		os.Exit(1)
	}

	var dataSlices [][]byte

	if *dirPath != "" {
		var paths []string
		err := filepath.WalkDir(*dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				paths = append(paths, path)
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking directory %s: %v\n", *dirPath, err)
			os.Exit(1)
		}

		sort.Strings(paths)

		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", path, err)
				os.Exit(1)
			}
			dataSlices = append(dataSlices, raw)
		}
	} else {
		raw, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", *inputFile, err)
			os.Exit(1)
		}
		parts := strings.Split(string(raw), *delimiter)
		for _, part := range parts {
			if part == "" {
				continue
			}
			dataSlices = append(dataSlices, []byte(part))
		}
	}

	treeJSON := merkle.BuildMarkle(dataSlices)

	if err := os.WriteFile(*outputFile, treeJSON, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Merkle tree written to %s\n", *outputFile)
}
