package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var getFilesCmd = &cobra.Command{
	Use:           "files",
	Short:         "Get all the uploaded file",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := http.Get(serverURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var files []struct {
			MerkleRoot string `json:"merkle_root"`
			FileName   string `json:"file_name"`
			UploadedAt string `json:"uploaded_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MERKLE ROOT\tFILE NAME\tUPLOADED AT")
		fmt.Fprintln(w, "------------\t---------\t-----------")
		for _, f := range files {
			fmt.Fprintf(w, "%s\t%s\t%s\n", f.MerkleRoot, f.FileName, f.UploadedAt)
		}
		w.Flush()

		return nil
	},
}

func init() {
	getFilesCmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "File server URL")
}
