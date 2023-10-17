package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
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
	http.HandleFunc("/api/mail/file", handleFileMail)
	http.HandleFunc("/api/archive/information", handleArchiveInformation)
	http.HandleFunc("/api/archive/files", handleAddFilesArchive)
	http.ListenAndServe(":8080", nil)
}

func handleFileMail(w http.ResponseWriter, r *http.Request) {

	file, header, err := r.FormFile("file")
	if err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes := bytes.Buffer{}
	_, err = fileBytes.ReadFrom(file)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	emails := r.FormValue("emails")

	err = sendMailWithAttachment(emails, header.Filename, fileBytes.Bytes())
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func sendMailWithAttachment(recipients, filename string, fileData []byte) error {
	from := os.Getenv("EMAIL_USERNAME")
	password := os.Getenv("EMAIL_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, smtpHost)

	message := []byte(fmt.Sprintf("Subject: File %s\nMIME-Version: 1.0\nContent-Type: multipart/mixed; boundary=MailBoundary\n\n--MailBoundary\nContent-Type: text/plain; charset=\"utf-8\"\n\nFile is attached.\n\n--MailBoundary\nContent-Disposition: attachment; filename=%s\nContent-Transfer-Encoding: base64\n\n%s\n--MailBoundary--", filename, filename, fileData))

	err := smtp.SendMail(fmt.Sprintf("%s:%s", smtpHost, smtpPort), auth, from, parseRecipients(recipients), message)
	if err != nil {

		return err
	}
	return nil
}

func parseRecipients(recipients string) []string {
	emails := []string{}
	for _, email := range splitEmails(recipients) {
		emails = append(emails, email)
	}
	return emails
}

func splitEmails(emails string) []string {

	return []string{emails}
}

func handleAddFilesArchive(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var validFiles []*multipart.FileHeader
	for _, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			contentType := fileHeader.Header.Get("Content-Type")
			if !isValidContentType(contentType) {
				http.Error(w, "Недопустимый формат файла: "+fileHeader.Filename, http.StatusBadRequest)
				return
			}
			validFiles = append(validFiles, fileHeader)
		}
	}

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)
	for _, fileHeader := range validFiles {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		zipFile, err := zipWriter.Create(fileHeader.Filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = io.Copy(zipFile, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = zipWriter.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	_, err = w.Write(zipBuffer.Bytes())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func isValidContentType(contentType string) bool {
	validTypes := map[string]bool{
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/xml": true,
		"image/jpeg":      true,
		"image/png":       true,
	}
	return validTypes[contentType]
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

		if strings.HasPrefix(filePath, "__MACOSX") {
			continue
		}

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
