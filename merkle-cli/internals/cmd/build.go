package cmd

import (
	"fmt"
	"os"

	"github.com/ah-naf/merkle-cli/internals/merkle"
	"github.com/ah-naf/merkle-cli/internals/utils"
	"github.com/spf13/cobra"
)

var (
	inFile    string
	dirPath   string
	outTree   string
	delimiter string
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a Merkle tree",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := utils.CollectData(inFile, dirPath, delimiter)
		if err != nil {
			return err
		}
		treeJSON := merkle.BuildMarkle(data)
		if err := os.WriteFile(outTree, treeJSON, 0o644); err != nil {
			return fmt.Errorf("write tree file: %w", err)
		}
		fmt.Printf("✅ Merkle tree written to %s\n", outTree)
		return nil
	},
}

func init() {
	buildCmd.Flags().StringVar(&inFile, "in", "", "Input file (mutually exclusive with --dir)")
	buildCmd.Flags().StringVar(&dirPath, "dir", "", "Input directory (mutually exclusive with --in)")
	buildCmd.Flags().StringVar(&outTree, "out", "tree.json", "Output Merkle tree JSON path")
	buildCmd.Flags().StringVar(&delimiter, "delimiter", "\n", "Record delimiter for input file")
	buildCmd.MarkFlagsMutuallyExclusive("in", "dir")
}
