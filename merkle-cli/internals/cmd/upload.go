package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/ah-naf/merkle-cli/internals/merkle"
	"github.com/spf13/cobra"
)

var (
	uploadFile string
	serverURL  string
	chunkSize  int
	timeout    time.Duration
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Resumably upload a file via Merkle-tree integrity to a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(uploadFile)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("stat file: %w", err)
		}
		totalSize := info.Size()
		totalChunks := int((totalSize + int64(chunkSize) - 1) / int64(chunkSize))
		fmt.Println(totalSize, totalChunks, chunkSize)

		// 2. Read all chunks into memory (for proof generation)
		chunks := make([][]byte, totalChunks)
		for i := 0; i < totalChunks; i++ {
			buf := make([]byte, chunkSize)
			n, err := io.ReadFull(f, buf)
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				buf = buf[:n]
			} else if err != nil {
				return fmt.Errorf("read chunk %d: %w", i, err)
			}
			chunks[i] = buf
		}

		// 3. Build Merkle tree client-side
		treeJSON := merkle.BuildMarkle(chunks)
		var rootNode merkle.Node
		if err := json.Unmarshal(treeJSON, &rootNode); err != nil {
			return fmt.Errorf("unmarshal tree JSON: %w", err)
		}
		merkleRoot := rootNode.Root

		// 4. POST /uploads/init
		initReq := map[string]interface{}{
			"merkle_root":  merkleRoot,
			"file_name":    filepath.Base(uploadFile),
			"file_size":    totalSize,
			"chunk_size":   chunkSize,
			"total_chunks": totalChunks,
		}

		body, _ := json.Marshal(initReq)
		client := &http.Client{Timeout: timeout}
		resp, err := client.Post(serverURL+"/uploads/init", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("init request: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("init failed: %s", respBody)
		}

		// 5. Loop: status → upload missing → repeat
		for {
			// GET /uploads/:root/status
			statusURL := fmt.Sprintf("%s/uploads/%s/status", serverURL, merkleRoot)
			resp, err := client.Get(statusURL)
			if err != nil {
				return fmt.Errorf("status request: %w", err)
			}
			var st struct {
				Uploaded []int `json:"uploaded"`
				Missing  []int `json:"missing"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
				return fmt.Errorf("decode status: %w", err)
			}
			resp.Body.Close()

			if len(st.Missing) == 0 {
				break // all done
			}

			missingChan := make(chan int, len(st.Missing))
			errChan := make(chan error, len(st.Missing))

			for _, idx := range st.Missing {
				missingChan <- idx
			}
			close(missingChan)

			concurrency := 4
			if s := os.Getenv("NUMBER_OF_WORKERS"); s != "" {
				if n, err := strconv.Atoi(s); err == nil && n > 0 {
					concurrency = n
				} else {
					fmt.Fprintf(os.Stderr, "WARNING: invalid NUMBER_OF_WORKERS=%q, using %d\n", s, concurrency)
				}
			}

			var wg sync.WaitGroup
			wg.Add(concurrency)

			for w := 0; w < concurrency; w++ {
				go func() {
					defer wg.Done()
					for idx := range missingChan {
						chunk := chunks[idx]

						// compute leaf & proof (rootNode is read‑only)
						sum := sha256.Sum256(chunk)
						leafHex := hex.EncodeToString(sum[:])
						proof, ok := merkle.GenerateProof(&rootNode, leafHex)
						if !ok {
							errChan <- fmt.Errorf("proof failed for chunk %d", idx)
							return
						}
						proofJSON, _ := json.Marshal(proof)

						// build request
						url := fmt.Sprintf("%s/uploads/%s/chunks/%d", serverURL, merkleRoot, idx)
						req, _ := http.NewRequest("PUT", url, bytes.NewReader(chunk))
						req.Header.Set("X-Leaf-Hash", leafHex)
						req.Header.Set("X-Merkle-Proof", string(proofJSON))

						// execute
						resp, err := client.Do(req)
						if err != nil {
							errChan <- fmt.Errorf("upload chunk %d: %w", idx, err)
							return
						}
						resp.Body.Close()
						if resp.StatusCode != http.StatusOK {
							errChan <- fmt.Errorf("chunk %d rejected: %s", idx, resp.Status)
							return
						}

						fmt.Printf("✓ uploaded chunk %d/%d\n", idx+1, totalChunks)
					}
				}()
			}
			go func() {
				wg.Wait()
				close(errChan)
			}()

			if err := <-errChan; err != nil {
				return err
			}
		}

		// 6. Finalize
		finalizeURL := fmt.Sprintf("%s/uploads/%s/finalize", serverURL, merkleRoot)
		req, _ := http.NewRequest("POST", finalizeURL, nil)
		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("finalize request: %w", err)
		}
		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			return fmt.Errorf("finalize failed: %s", body)
		}
		fmt.Println("🎉 upload complete! Merkle root:", merkleRoot)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVar(&uploadFile, "file", "", "Path to file to upload")
	uploadCmd.MarkFlagRequired("file")
	uploadCmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Upload server URL")
	uploadCmd.Flags().IntVar(&chunkSize, "chunk-size", 1<<20, "Chunk size in bytes")
	uploadCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "HTTP client timeout")
}
