package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ah-naf/merkle-cli/internals/merkle"
	"github.com/ah-naf/merkle-cli/internals/utils"
	"github.com/spf13/cobra"
)

var (
	treeFile   string
	targetFile string
	proofOut   string
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify inclusion of a file in the Merkle tree",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := utils.LoadTree(treeFile)
		if err != nil {
			return err
		}
		leafHash, err := utils.FileHash(targetFile)
		if err != nil {
			return err
		}
		proof, found := merkle.GenerateProof(root, leafHash)
		if !found {
			return fmt.Errorf("proof invalid for %s (leaf %s)", targetFile, leafHash)
		}

		if proofOut != "" {
			proofJSON, err := json.MarshalIndent(proof, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal proof JSON: %w", err)
			}
			if err := os.WriteFile(proofOut, proofJSON, 0o644); err != nil {
				return fmt.Errorf("write proof file: %w", err)
			}
			fmt.Printf("ℹ️ Merkle proof written to %s\n", proofOut)
		}

		if !merkle.VerifyProof(*proof) {
			return fmt.Errorf("proof invalid for %s (leaf %s)", targetFile, leafHash)
		}
		fmt.Printf("✔️ Proof valid for %s (leaf %s)\n", targetFile, leafHash)
		return nil
	},
}

func init() {
	verifyCmd.Flags().StringVar(&treeFile, "tree", "", "Merkle tree JSON file path")
	verifyCmd.Flags().StringVar(&targetFile, "file", "", "File path to verify inclusion for")
	verifyCmd.Flags().StringVar(&proofOut, "out-proof", "", "Optional path to write proof JSON")
	verifyCmd.MarkFlagRequired("tree")
	verifyCmd.MarkFlagRequired("file")
}
