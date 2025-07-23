package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type FileInfo struct {
	MerkleRoot string `json:"merkle_root"`
	FileName   string `json:"file_name"`
	UploadedAt string `json:"uploaded_at"`
}

func (h *UploadHandler) GetFiles(c *gin.Context) {
	rows, err := h.db.Query("SELECT merkle_root, file_name, completed_at FROM uploads")
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var files []FileInfo

	for rows.Next() {
		var file FileInfo
		if err := rows.Scan(&file.MerkleRoot, &file.FileName, &file.UploadedAt); err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		files = append(files, file)
	}
	c.JSON(http.StatusOK, files)
}
