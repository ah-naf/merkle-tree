package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ah-naf/merkle-upload-service/internal/db"
	"github.com/ah-naf/merkle-upload-service/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	db.Init()

	router := gin.Default()

	router.GET("/health", func(ctx *gin.Context) {
		if err := db.DB.Ping(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	h := handler.NewUploadHandler(db.DB, os.Getenv("STORAGE_PATH"))
	r := gin.Default()
	r.POST("/uploads/init", h.InitUpload)
	r.PUT("/uploads/:root/chunks/:idx", h.PutChunk)
	r.GET("/uploads/:root/status", h.Status)
	r.POST("/uploads/:root/finalize", h.Finalize)

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
