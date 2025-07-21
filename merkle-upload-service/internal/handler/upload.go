package handler

import (
	"database/sql"
	"net/http"

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
			c.JSON(http.StatusConflict, gin.H{"error": "upload already initialized"})
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
