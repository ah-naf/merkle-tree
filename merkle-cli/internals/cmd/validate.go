package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [merkle_root]",
	Short: "Verify server-side chunks & DB match the merkle root",
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("%s/verify/%s", serverURL, root)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("validation failed: %v", result)
		}
		fmt.Println("✅ validation OK:", result)
		return nil
	},
}

func init() {
	validateCmd.Flags().StringVar(&root, "root", "", "Hash of the file that you want to validate")
	validateCmd.MarkFlagRequired("root")
}
