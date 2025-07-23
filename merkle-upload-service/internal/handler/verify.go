package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ah-naf/merkle-upload-service/internal/merkle"
	"github.com/gin-gonic/gin"
)

// Verify: GET /verify/:root
//
// 1) Fetch all chunk hashes from DB and filesystem.
// 2) Recompute Merkle root client‑side.
// 3) Compare to stored root in uploads table.
func (h *UploadHandler) Verify(c *gin.Context) {
	root := c.Param("root")

	var chunkSize, totalChunks int
	err := h.db.QueryRow(
		`SELECT chunk_size, total_chunks FROM uploads WHERE merkle_root=$1`,
		root,
	).Scan(&chunkSize, &totalChunks)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		return
	}

	dir := filepath.Join(h.storagePath, root)
	chunks := make([][]byte, totalChunks)
	for i := 0; i < totalChunks; i++ {
		raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%d.chunk", i)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("chunk %d missing", i)})
			return
		}

		// compare raw hash vs DB record
		var dbHash string
		_ = h.db.QueryRow(
			`SELECT leaf_hash FROM upload_chunks WHERE merkle_root=$1 AND chunk_index=$2`,
			root, i,
		).Scan(&dbHash)
		actual := sha256.Sum256(raw)
		if hex.EncodeToString(actual[:]) != dbHash {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("hash mismatch at chunk %d", i)})
			return
		}
		chunks[i] = raw
	}

	treeJSON := merkle.BuildMarkle(chunks)
	var node merkle.Node
	if err := json.Unmarshal(treeJSON, &node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not unmarshal tree"})
		return
	}
	if node.Root != root {
		c.JSON(http.StatusConflict, gin.H{
			"status":     "invalid",
			"expected":   root,
			"recomputed": node.Root,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "merkle_root": root})
}
