package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	root    string
	outPath string
)

var downloadCmd = &cobra.Command{
	Use:           "download [merkle_root] [output_file]",
	Short:         "Verify and download & merge all chunks",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("%s/download/%s", serverURL, root)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)

			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err != nil {
				return fmt.Errorf("download failed: %s", string(bodyBytes))
			}
			return fmt.Errorf("download failed:\n%s", prettyJSON.String())
		}

		f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			"downloading",
		)

		if _, err := io.Copy(io.MultiWriter(f, bar), resp.Body); err != nil {
			return err
		}

		fmt.Println("\n✅ Download complete!")
		return nil
	},
}

func init() {
	downloadCmd.Flags().StringVar(&root, "root", "", "Hash of the file that you want to download")
	downloadCmd.MarkFlagRequired("root")
	downloadCmd.Flags().StringVar(&outPath, "out", "", "Download path")
	downloadCmd.MarkFlagRequired("out")
	downloadCmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Upload server URL")
	downloadCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "HTTP client timeout")
}
