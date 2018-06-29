package main

import (
	"flag"
	"log"
	"mime"
	"path/filepath"
	"sort"
	"strings"
	"time"

	marionette "github.com/njasm/marionette_client"
)

func main() {
	log.SetFlags(log.Lshortfile)
	var (
		team = flag.String("team", "", "Slack team to download emojis for")
		dir  = flag.String("i", "", "Directory to import emoji from")
		from = flag.Int("from", 1, "Index to start from")
	)
	flag.Parse()

	if *team == "" {
		log.Fatal("must specify a team")
	}
	if *dir == "" {
		log.Fatal("must specify an input directory")
	}

	mc := marionette.NewClient()
	if err := mc.Connect("", 0); err != nil {
		log.Fatal(err)
	}
	if _, err := mc.NewSession("", nil); err != nil {
		log.Fatalf("failed to create new session: %s", err)
	}

	mc.Navigate(getBaseUrl(*team))

	images := findImages(*dir)
	for i, image := range images {
		if i+1 < *from {
			continue
		}
		log.Printf("[%d/%d] Uploading %s\n", i+1, len(images), image)
		upload(mc, image)
	}
}

func getBaseUrl(team string) string {
	return "https://" + team + ".slack.com/customize/emoji"
}

func findImages(dir string) []string {
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	images := make([]string, 0, len(files))
	for _, file := range files {
		mimeType := mime.TypeByExtension(filepath.Ext(file))
		if strings.HasPrefix(mimeType, "image/") {
			images = append(images, file)
		}
	}
	sort.Strings(images)
	return images
}

func waitForElement(finder marionette.Finder, id string) *marionette.WebElement {
	_, elem, err := marionette.Wait(finder).For(30 * time.Second).Until(marionette.ElementIsPresent(marionette.ID, id))
	if err != nil {
		log.Fatal(err)
	}
	return elem
}

func upload(mc *marionette.Client, image string) {
	nameInput := waitForElement(mc, "emojiname")

	name := filepath.Base(image)
	name = name[:len(name)-len(filepath.Ext(name))]
	nameInput.Clear()
	if err := nameInput.SendKeys(name); err != nil {
		log.Fatal(err)
	}

	uploadInput := waitForElement(mc, "emojiimg")

	abs, err := filepath.Abs(image)
	if err != nil {
		log.Fatal(err)
	}
	if err := uploadInput.SendKeys(abs); err != nil {
		log.Fatal(err)
	}

	submit := waitForElement(mc, "addemoji_submit")
	submit.Click()

	marionette.Wait(mc).Until(marionette.ElementIsNotPresent(marionette.ID, "addemoji_submit"))
}
