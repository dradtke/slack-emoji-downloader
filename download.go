package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/njasm/marionette_client"
)

func main() {
	log.SetFlags(log.Lshortfile)
	var (
		team = flag.String("team", "", "Slack team to download emojis for")
		dir  = flag.String("o", "", "Directory to save emoji in")
		from = flag.Int("from", 1, "Page number to start from")
	)
	flag.Parse()

	if *team == "" {
		log.Fatal("must specify a team")
	}
	if *dir == "" {
		log.Fatal("must specify an output directory")
	}

	client := marionette_client.NewClient()
	if err := client.Connect("", 0); err != nil {
		log.Fatal(err)
	}
	if _, err := client.NewSession("", nil); err != nil {
		log.Fatalf("failed to create new session: %s", err)
	}

	baseUrl := getBaseUrl(*team)

	for page := *from; true; page++ {
		log.Printf("Loading page %d...", page)
		client.Navigate(fmt.Sprintf("%s?page=%d", baseUrl, page))
		table, err := client.FindElement(marionette_client.ID, "custom_emoji")
		if err != nil {
			log.Print("Done!")
		}
		findEmoji(table, *dir)
	}

	/*
		table, err := client.FindElement(marionette_client.ID, "custom_emoji")
		if err != nil {
			log.Fatal(err)
		}
		findEmoji(table, *dir)
	*/
}

func getBaseUrl(team string) string {
	return "https://" + team + ".slack.com/customize/emoji"
}

func findEmoji(table *marionette_client.WebElement, dir string) {
	rows, err := table.FindElements(marionette_client.CLASS_NAME, "emoji_row")
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(rows))
	for _, row := range rows {
		name, url := parseEmojiRow(row)
		filename := filepath.Join(dir, name) + filepath.Ext(url)
		go func(filename, url string) {
			save(filename, url)
			wg.Done()
		}(filename, url)
	}
	wg.Wait()
}

func save(filename, url string) {
	if url == "" {
		return
	}
	var src io.Reader
	if strings.HasPrefix(url, "data:") {
		semicolon := strings.Index(url, ";")
		end := strings.Index(url, "base64,") + len("base64,")

		format := url[len("data:"):semicolon]
		data := url[end:]

		formatParts := strings.Split(format, "/")
		if len(formatParts) != 2 || formatParts[0] != "image" {
			log.Fatal("invalid format: " + format)
		}
		filename += "." + formatParts[1]

		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			log.Fatal(err)
		}
		src = bytes.NewReader(b)
	} else {
		// For HTTP urls, abort early if the file already exists.
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			log.Println("Found " + filename)
			return
		}
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		src = resp.Body
		defer resp.Body.Close()
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		log.Println("Saved " + filename)
	}()

	if _, err := io.Copy(f, src); err != nil {
		log.Fatal(err)
	}
}

func parseEmojiRow(row *marionette_client.WebElement) (name, url string) {
	cols, err := row.FindElements(marionette_client.TAG_NAME, "td")
	if err != nil {
		log.Fatal(err)
	}
	for _, col := range cols {
		switch col.Attribute("headers") {
		case "custom_emoji_image":
			wrapper, err := col.FindElement(marionette_client.CLASS_NAME, "emoji-wrapper")
			if err != nil {
				log.Fatal(err)
			}
			url = wrapper.Attribute("data-original")
		case "custom_emoji_name":
			name = col.Text()
			name = name[1:strings.LastIndex(name, ":")]
		}
	}
	return
}

func do(resp *marionette_client.Response, err error) string {
	if err != nil {
		log.Fatal(err)
	}
	return resp.Value
}
