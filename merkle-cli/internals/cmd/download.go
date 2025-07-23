package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	root    string
	outPath string
)

var downloadCmd = &cobra.Command{
	Use:   "download [merkle_root] [output_file]",
	Short: "Verify and download & merge all chunks",
	RunE: func(cmd *cobra.Command, args []string) error {
		// first, verify on server
		verifyURL := fmt.Sprintf("%s/uploads/%s/verify", serverURL, root)
		vr, err := http.Get(verifyURL)
		if err != nil {
			return err
		}
		if vr.StatusCode != http.StatusOK {
			var detail map[string]interface{}
			_ = json.NewDecoder(vr.Body).Decode(&detail)
			return fmt.Errorf("verification failed: %v", detail)
		}
		vr.Body.Close()

		// then download merged file
		url := fmt.Sprintf("%s/uploads/%s/download", serverURL, root)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("download failed: %s", string(body))
		}

		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return err
		}

		fmt.Println("✅ downloaded to", outPath)
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
