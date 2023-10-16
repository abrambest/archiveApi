package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type File struct {
	FilePath string  `json:"file_path"`
	Size     float64 `json:"size"`
	MimeType string  `json:"mimetype"`
}

type Response struct {
	FileName    string  `json:"filename"`
	ArchiveSize float64 `json:"archive_size"`
	TotalSize   float64 `json:"total_size"`
	TotalFiles  float64 `json:"total_files"`
	Files       []File  `json:"files"`
}

func main() {
	http.HandleFunc("/api/archive/information", handleArchiveInformation)
	http.ListenAndServe(":8080", nil)
}

func handleArchiveInformation(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Header.Get("Content-Type") != "application/zip" {
		http.Error(w, "Invalid file format. Please upload a valid zip file.", http.StatusBadRequest)
		return
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, file)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	rBuf, err := zip.NewReader(strings.NewReader(buf.String()), int64(buf.Len()))
	if err != nil {
		http.Error(w, "Invalid zip file", http.StatusBadRequest)
		return
	}

	var totalSize float64
	var files []File

	for _, f := range rBuf.File {
		filePath := f.Name
		size := float64(f.UncompressedSize64)

		totalSize += size
		mimeType := ""

		if strings.HasSuffix(filePath, ".docx") {
			mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

		} else {
			rc, err := f.Open()
			if err != nil {
				fmt.Println("Error opening file:", err)

			}
			defer rc.Close()

			buffer := make([]byte, 512)
			_, err = rc.Read(buffer)
			if err != nil && err != io.EOF {
				log.Println("Error reading file from zip:", err)
				continue
			}

			mimeType = http.DetectContentType(buffer)
		}

		files = append(files, File{
			FilePath: filePath,
			Size:     size,
			MimeType: mimeType,
		})
	}

	resp := Response{
		FileName:    header.Filename,
		ArchiveSize: float64(buf.Len()),
		TotalSize:   totalSize,
		TotalFiles:  float64(len(files)),
		Files:       files,
	}

	responseJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
