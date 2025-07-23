package handler

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ah-naf/merkle-upload-service/internal/merkle"
	"github.com/ah-naf/merkle-upload-service/internal/model"
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	db          *sql.DB
	storagePath string
}

func NewUploadHandler(db *sql.DB, storagePath string) *UploadHandler {
	return &UploadHandler{db: db, storagePath: storagePath}
}

// POST /uploads/init
func (h *UploadHandler) InitUpload(c *gin.Context) {
	var req model.InitUploadRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	var exists int
	err := h.db.QueryRow("SELECT 1 FROM uploads WHERE merkle_root=$1", req.MerkleRoot).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			c.JSON(http.StatusCreated, gin.H{"message": "upload already initialized"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// create the upload session
	_, err = h.db.Exec(
		`INSERT INTO uploads
           (merkle_root, file_name, file_size, chunk_size, total_chunks)
         VALUES ($1,$2,$3,$4,$5)`,
		req.MerkleRoot, req.FileName, req.FileSize, req.ChunkSize, req.TotalChunks,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, model.InitUploadResponse{
		MerkleRoot:  req.MerkleRoot,
		TotalChunks: req.TotalChunks,
		Uploaded:    []int{},
	})
}

// PUT /uploads/:root/chunks/:idx
// Expects:
//   - raw chunk bytes in body
//   - header   X-Leaf-Hash: "<hex Li>"
//   - header   X-Merkle-Proof: JSON of []ProofStep (from client)
//
// Verifies SHA256(chunk)==Li and merkle.VerifyProof(proof) before storing.
func (h *UploadHandler) PutChunk(c *gin.Context) {
	root := c.Param("root")
	idx, err := strconv.Atoi(c.Param("idx"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk index"})
		return
	}

	// read raw chunk
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read body"})
		return
	}

	// 1) verify the chunk’s own hash
	leaf := sha256.Sum256(data)
	leafHex := hex.EncodeToString(leaf[:])
	if leafHex != c.GetHeader("X-Leaf-Hash") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "leaf-hash mismatch"})
		return
	}

	// 2) parse & verify the client’s Merkle proof
	var proof merkle.MerkleProof
	if err := json.Unmarshal([]byte(c.GetHeader("X-Merkle-Proof")), &proof); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proof JSON"})
		return
	}
	// ensure the proof’s root matches our session’s root
	if proof.Root != root {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proof root mismatch"})
		return
	}
	if !merkle.VerifyProof(proof) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "merkle proof failed"})
		return
	}

	// 3) record in DB (idempotent)
	if _, err := h.db.Exec(
		`INSERT INTO upload_chunks(merkle_root,chunk_index,leaf_hash)
         VALUES($1,$2,$3) ON CONFLICT DO NOTHING`,
		root, idx, leafHex,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4) save chunk to local storage
	dir := filepath.Join(h.storagePath, root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create storage dir"})
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.chunk", idx))
	if err := os.WriteFile(path, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not write chunk"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "chunk ok", "chunk_index": idx})
}

// GET /uploads/:root/status
// Returns which indices have been stored so the client can retry only the rest.
func (h *UploadHandler) Status(c *gin.Context) {
	root := c.Param("root")

	// fetch total_chunks
	var total int
	if err := h.db.QueryRow(
		`SELECT total_chunks FROM uploads WHERE merkle_root=$1`,
		root,
	).Scan(&total); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	rows, err := h.db.Query(
		`SELECT chunk_index FROM upload_chunks WHERE merkle_root=$1`,
		root,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	have := make([]bool, total)
	for rows.Next() {
		var i int
		rows.Scan(&i)
		if i >= 0 && i < total {
			have[i] = true
		}
	}

	uploaded, missing := []int{}, []int{}
	for i := 0; i < total; i++ {
		if have[i] {
			uploaded = append(uploaded, i)
		} else {
			missing = append(missing, i)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"uploaded": uploaded,
		"missing":  missing,
	})
}

// POST /uploads/:root/finalize
// Marks the upload complete (we trust the per-chunk proofs).
func (h *UploadHandler) Finalize(c *gin.Context) {
	root := c.Param("root")

	// ensure the session exists
	res, err := h.db.Exec(
		`UPDATE uploads SET completed_at=now() WHERE merkle_root=$1`,
		root,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "completed",
		"merkle_root": root,
	})
}

// Verify: GET /uploads/:root/verify
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

// Download: GET /uploads/:root/download
func (h *UploadHandler) Download(c *gin.Context) {
	root := c.Param("root")

	verifyURL := fmt.Sprintf("%s/uploads/%s/verify", h.baseURL(), root)
	resp, err := http.Get(verifyURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification failed", "detail": errResp})
		return
	}
	resp.Body.Close()

	var total int
	if err := h.db.QueryRow(
		"SELECT total_chunks FROM uploads WHERE merkle_root=$1", root,
	).Scan(&total); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		return
	}

	fmt.Println("root", root)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.bin"`, root))
	c.Header("Content-Type", "application/octet-stream")

	dir := filepath.Join(h.storagePath, root)
	for idx := 0; idx < total; idx++ {
		path := filepath.Join(dir, fmt.Sprintf("%d.chunk", idx))
		f, err := os.Open(path)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "cannot open chunk", "index": idx})
			return
		}
		if _, err := io.Copy(c.Writer, f); err != nil {
			f.Close()
			return
		}
		f.Close()
	}
}

func (h *UploadHandler) baseURL() string {
	return os.Getenv("SERVER_URL")
}
