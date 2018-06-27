package main

import (
	"flag"
	"log"
	"mime"
	"path/filepath"
	"strings"

	"github.com/njasm/marionette_client"
)

func main() {
	log.SetFlags(log.Lshortfile)
	var (
		team = flag.String("team", "", "Slack team to download emojis for")
		dir  = flag.String("i", "", "Directory to import emoji from")
	)
	flag.Parse()

	if *team == "" {
		log.Fatal("must specify a team")
	}
	if *dir == "" {
		log.Fatal("must specify an input directory")
	}

	client := marionette_client.NewClient()
	if err := client.Connect("", 0); err != nil {
		log.Fatal(err)
	}
	if _, err := client.NewSession("", nil); err != nil {
		log.Fatalf("failed to create new session: %s", err)
	}

	client.Navigate(getBaseUrl(*team))

	images := findImages(*dir)
	for i, image := range images {
		log.Printf("[%d/%d] Uploading %s\n", i+1, len(images), image)
		upload(client, image)
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
	return images
}

func isImage(image string) bool {
	mimeType := mime.TypeByExtension(filepath.Ext(image))
	return strings.HasPrefix(mimeType, "image/")
}

func upload(client *marionette_client.Client, image string) {
	nameInput, err := client.FindElement(marionette_client.ID, "emojiname")
	if err != nil {
		log.Fatal(err)
	}
	name := filepath.Base(image)
	name = name[:len(name)-len(filepath.Ext(name))]
	nameInput.Clear()
	if err := nameInput.SendKeys(name); err != nil {
		log.Fatal(err)
	}

	uploadInput, err := client.FindElement(marionette_client.ID, "emojiimg")
	if err != nil {
		log.Fatal(err)
	}
	abs, err := filepath.Abs(image)
	if err != nil {
		log.Fatal(err)
	}
	if err := uploadInput.SendKeys(abs); err != nil {
		log.Fatal(err)
	}

	submit, err := client.FindElement(marionette_client.ID, "addemoji_submit")
	if err != nil {
		log.Fatal(err)
	}
	submit.Click()
}
