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
	"time"

	marionette "github.com/njasm/marionette_client"
)

func main() {
	log.SetFlags(log.Lshortfile)
	var (
		team = flag.String("team", "", "Slack team to download emojis for")
		dir  = flag.String("o", "", "Directory to save emoji in")
	)
	flag.Parse()

	if *team == "" {
		log.Fatal("must specify a team")
	}
	if *dir == "" {
		log.Fatal("must specify an output directory")
	}
	if err := os.Chdir(*dir); err != nil {
		log.Fatalf("failed to cd to output directory: %s", err)
	}

	mc := marionette.NewClient()
	if err := mc.Connect("", 0); err != nil {
		log.Fatal(err)
	}
	if _, err := mc.NewSession("", nil); err != nil {
		log.Fatalf("failed to create new session: %s", err)
	}

	mc.Navigate(getEmojiUrl(*team))

	table := waitForQaElement(mc, "customize_emoji_table", 10*time.Second)
	navigableTable, err := table.FindElement(marionette.CLASS_NAME, "c-table_view_keyboard_navigable_container")
	if err != nil {
		log.Fatalf("failed to find navigable table: %s", err)
	}

	for {
		downloadAvailable(navigableTable)
		if err := navigableTable.SendKeys("\ue00f"); err != nil {
			log.Fatalf("failed to send page down to table: %s", err)
		}
		time.Sleep(1 * time.Second)
	}
}

func downloadAvailable(f marionette.Finder) {
	for _, row := range findQaElements(f, "virtual-list-item") {
		imgEl, err := row.FindElement(marionette.CLASS_NAME, "p-customize_emoji_list__image")
		if err != nil {
			log.Fatalf("failed to find image element: %s", err)
		}
		name, url := imgEl.Attribute("alt"), imgEl.Attribute("src")
		fmt.Printf("downloading %s\n", name)
		filename := name + filepath.Ext(url)
		save(filename, url)
	}
}

func waitForQaElement(f marionette.Finder, name string, timeout time.Duration) *marionette.WebElement {
	condition := marionette.ElementIsPresent(marionette.CSS_SELECTOR, dataQaSelector(name))
	ok, el, err := marionette.Wait(f).For(timeout).Until(condition)
	if err != nil {
		log.Fatalf("wait for %s failed: %s", name, err)
	} else if !ok {
		log.Fatal("wait for %s failed: not ok", name)
	}
	return el
}

func findQaElements(f marionette.Finder, name string) []*marionette.WebElement {
	els, err := f.FindElements(marionette.CSS_SELECTOR, dataQaSelector(name))
	if err != nil {
		log.Fatalf("failed to find elements under %s: %s", name, err)
	}
	return els
}

func dataQaSelector(name string) string {
	return fmt.Sprintf(`[data-qa="%s"]`, name)
}

func getEmojiUrl(team string) string {
	return "https://" + team + ".slack.com/customize/emoji"
}

/*
func findEmoji(client, dir string) {
	rows, err := table.FindElements(marionette.CLASS_NAME, "emoji_row")
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
*/

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

func parseEmojiRow(row *marionette.WebElement) (name, url string) {
	cols, err := row.FindElements(marionette.TAG_NAME, "td")
	if err != nil {
		log.Fatal(err)
	}
	for _, col := range cols {
		switch col.Attribute("headers") {
		case "custom_emoji_image":
			wrapper, err := col.FindElement(marionette.CLASS_NAME, "emoji-wrapper")
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

func do(resp *marionette.Response, err error) string {
	if err != nil {
		log.Fatal(err)
	}
	return resp.Value
}
