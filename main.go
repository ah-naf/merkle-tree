package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ah-naf/merkle-cli/merkle"
)

func main() {
	inputFile := flag.String("in", "", "Path to file with data blocks")
	outputFile := flag.String("out", "output.json", "Path to write Merkle tree JSON")
	delimiter := flag.String("delimiter", "\n", "Delimiter to split records in the input file")

	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --in is required")
		flag.Usage()
		os.Exit(1)
	}
	if *outputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --out is required")
		flag.Usage()
		os.Exit(1)
	}

	if *delimiter == "" {
		fmt.Fprintln(os.Stderr, "Error: --delimiter cannot be empty")
		flag.Usage()
		os.Exit(1)
	}

	raw, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", *inputFile, err)
		os.Exit(1)
	}

	parts := strings.Split(string(raw), *delimiter)

	var data [][]byte
	for _, part := range parts {
		if part == "" {
			continue
		}
		data = append(data, []byte(part))
	}

	merkleTree := merkle.BuildMarkle(data)

	if err := os.WriteFile(*outputFile, merkleTree, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *outputFile, err)
		os.Exit(1)
	}
}
