package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const metaFile = "files.json"

//go:embed index.html
var static embed.FS

type fileEntry struct {
	OriginalName string `json:"name"`
	Path         string `json:"-"` // reconstructed from ID + storageDir
	ID           string `json:"id"`
}

var (
	files   = make(map[string]fileEntry)
	filesMu sync.Mutex
)

func storageDir() string {
	if dir := os.Getenv("STORAGE_DIR"); dir != "" {
		return dir
	}
	return "data"
}

func metaPath() string {
	return filepath.Join(storageDir(), metaFile)
}

func saveFiles() {
	data, _ := json.Marshal(files)
	os.WriteFile(metaPath(), data, 0o644)
}

func loadFilesFromDisk() {
	data, err := os.ReadFile(metaPath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &files)
	for id, entry := range files {
		entry.Path = filepath.Join(storageDir(), id)
		files[id] = entry
	}
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, _ := static.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	id := generateID()
	dir := storageDir()
	os.MkdirAll(dir, 0o755)
	destPath := filepath.Join(dir, id)

	dest, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Failed to store file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to store file", http.StatusInternalServerError)
		return
	}

	entry := fileEntry{OriginalName: header.Filename, Path: destPath, ID: id}
	filesMu.Lock()
	files[id] = entry
	saveFiles()
	filesMu.Unlock()

	log.Printf("stored file %q as %s", header.Filename, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	filesMu.Lock()
	list := make([]fileEntry, 0, len(files))
	for _, entry := range files {
		list = append(list, entry)
	}
	filesMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/download/"):]
	if id == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}

	filesMu.Lock()
	entry, ok := files[id]
	if ok {
		delete(files, id)
		saveFiles()
	}
	filesMu.Unlock()

	if !ok {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(entry.Path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	defer os.Remove(entry.Path)

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, entry.OriginalName))
	io.Copy(w, f)
	log.Printf("served and deleted file %q (%s)", entry.OriginalName, id)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	os.MkdirAll(storageDir(), 0o755)
	loadFilesFromDisk()
	log.Printf("loaded %d file(s) from disk", len(files))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/files", filesHandler)
	http.HandleFunc("/download/", downloadHandler)

	log.Printf("filez listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
