package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
)

func makeItemFromFile(file fs.DirEntry) (feeds.Item, error) {
	title := file.Name()

	if len(title) < 1 {
		return feeds.Item{}, fmt.Errorf("No item title.")
	}

	url := os.Getenv("ITEM_URL")

	item := feeds.Item{
		Title: title,
		Link:  &feeds.Link{Href: url + "/" + title},
	}

	// Add enclosure for MP3 files
	if strings.HasSuffix(strings.ToLower(title), ".mp3") {
		// Get file size for the enclosure
		filePath := filepath.Join("./files/", title)
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Warning: Could not get file info for %s: %v", title, err)
		} else {
			// Use /files/ path for the enclosure URL to match our MP3 handler
			enclosureURL := strings.TrimSuffix(url, "/") + "/files/" + title
			item.Enclosure = &feeds.Enclosure{
				Url:    enclosureURL,
				Length: fmt.Sprintf("%d", fileInfo.Size()),
				Type:   "audio/mpeg",
			}
		}
	}

	return item, nil
}

func makeFeed(files []fs.DirEntry) (string, error) {
	feed := &feeds.Feed{
		Title:       os.Getenv("FEED_TITLE"),
		Link:        &feeds.Link{Href: os.Getenv("FEED_URL")},
		Description: os.Getenv("FEED_DESC"),
		Author:      &feeds.Author{Name: os.Getenv("FEED_AUTHOR")},
		Created:     time.Now(),
	}

	for _, file := range files {
		item, err := makeItemFromFile(file)
		if err != nil {
			return "", err
		}
		feed.Add(&item)
	}

	rss, err := feed.ToRss()
	if err != nil {
		return "", err
	}

	return rss, nil
}

func serveMP3Handler(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL path
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Construct file path
	filePath := filepath.Join("./files/", filename)

	// Check if file exists and is an MP3
	if !strings.HasSuffix(strings.ToLower(filename), ".mp3") {
		http.Error(w, "Only MP3 files are supported", http.StatusBadRequest)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Set appropriate headers for MP3 files
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")

	// Serve the file
	http.ServeFile(w, r, filePath)
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	// Read files directory dynamically on each request
	files, err := os.ReadDir("./files/")
	if err != nil {
		log.Printf("Error reading files directory: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate feed based on current files
	feed, err := makeFeed(files)
	if err != nil {
		log.Printf("Error generating feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/rss+xml")
	fmt.Fprint(w, feed)
}

func main() {
	// Check if files directory exists at startup
	if _, err := os.Stat("./files/"); os.IsNotExist(err) {
		log.Fatal("Files directory './files/' does not exist")
	}

	router := mux.NewRouter().StrictSlash(true)

	// Serve MP3 files specifically at /files/{filename}
	router.HandleFunc("/files/{filename:.+\\.mp3}", serveMP3Handler).Methods("GET")

	// Keep existing feed file server
	router.PathPrefix("/feed/").
		Handler(http.StripPrefix("/feed", http.FileServer(http.Dir("./files/"))))

	// Dynamic feed generation
	router.HandleFunc("/", feedHandler)

	HTTP_PORT := os.Getenv("HTTP_PORT")
	log.Println("HTTP server running on port: " + HTTP_PORT)
	http.ListenAndServe(":"+HTTP_PORT, router)
}
