package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "merkle",
	Short: "Merkle CLI",
	Long:  "A CLI tool for building and verifying Merkle trees.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(verifyCmd)
}
