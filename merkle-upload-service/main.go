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
	router.POST("/uploads/init", h.InitUpload)
	router.PUT("/uploads/:root/chunks/:idx", h.PutChunk)
	router.GET("/uploads/:root/status", h.Status)
	router.POST("/uploads/:root/finalize", h.Finalize)

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
