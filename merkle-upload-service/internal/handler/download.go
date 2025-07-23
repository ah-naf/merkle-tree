package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Download: GET /download/:root
func (h *UploadHandler) Download(c *gin.Context) {
	root := c.Param("root")

	verifyURL := fmt.Sprintf("%s/verfiy/%s", h.baseURL(), root)
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
