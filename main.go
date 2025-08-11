package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
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

	return feeds.Item{
		Title: title,
		Link:  &feeds.Link{Href: url + "/" + title},
	}, nil
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

func main() {
	files, err := os.ReadDir("./files/")
	if err != nil {
		log.Panic(err)
	}

	feed, err := makeFeed(files)
	if err != nil {
		log.Panic(err)
	}

	router := mux.NewRouter().StrictSlash(true)

	router.PathPrefix("/feed/").
		Handler(http.StripPrefix("/feed", http.FileServer(http.Dir("./files/"))))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, feed)
	})

	HTTP_PORT := os.Getenv("HTTP_PORT")
	log.Println("HTTP server running on port: " + HTTP_PORT)
	http.ListenAndServe(":"+HTTP_PORT, router)
}
