package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type FileDetails struct {
	Name string
	Size int64
}

func uploadFile(c *gin.Context) {
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		c.String(http.StatusInternalServerError, "Error parsing form")
		return
	}

	file, handler, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error getting file")
		return
	}
	defer file.Close()

	f, err := os.OpenFile(filepath.Join("uploads", handler.Filename), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating file")
		return
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error copying file")
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func listFiles() ([]FileDetails, error) {
	var files []FileDetails

	err := filepath.Walk("uploads", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, FileDetails{Name: info.Name(), Size: info.Size()})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func renderTemplate(c *gin.Context, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error parsing template")
		return
	}
	err = t.Execute(c.Writer, data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error executing template")
	}
}

func homePage(c *gin.Context) {
	files, err := listFiles()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error listing files")
		return
	}

	renderTemplate(c, "index.html", files)
}

func streamVideo(c *gin.Context) {
	filename := c.Query("file")
	if filename == "" {
		c.String(http.StatusBadRequest, "File parameter missing")
		return
	}

	filePath := filepath.Join("uploads", filename)
	f, err := os.Open(filePath)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error opening file")
		return
	}
	defer f.Close()

	contentType := "video/mp4"
	if filepath.Ext(filePath) == ".webm" {
		contentType = "video/webm"
	}

	stat, err := f.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error getting file info")
		return
	}
	fileSize := strconv.FormatInt(stat.Size(), 10)

	c.Header("Content-Disposition", "inline")
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fileSize)
	c.Header("Accept-Ranges", "bytes")

	// Serve the file
	http.ServeContent(c.Writer, c.Request, filename, time.Now(), f)
}

func main() {

	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		os.Mkdir("uploads", os.ModePerm)
	}

	router := gin.Default()

	router.Static("/static", "./static")

	router.GET("/", homePage)
	router.POST("/upload", uploadFile)
	router.GET("/stream", streamVideo)

	// Start the HTTP server
	fmt.Println("Server listening on port 8080...")
	router.Run(":8080")
}
