package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/slack-go/slack"
)

func main() {
	log.SetFlags(log.Lshortfile)
	var (
		dir     = flag.String("output", "", "Directory to save emoji in")
		token   = flag.String("token", "", "Slack API token")
		workers = flag.Int("workers", 4, "number of workers")
	)
	flag.Parse()

	if *dir == "" {
		log.Fatal("must specify an output directory")
	}
	if *token == "" {
		log.Fatal("must specify an API token")
	}

	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		if err := os.MkdirAll(*dir, 0755); err != nil {
			log.Fatalf("failed to create output directory: %s", err)
		}
	}
	if err := os.Chdir(*dir); err != nil {
		log.Fatalf("failed to cd to output directory: %s", err)
	}

	retrieveEmoji(slack.New(*token), *workers)
}

type emojiData struct {
	name, url string
}

func retrieveEmoji(s *slack.Client, workers int) {
	log.Println("fetching emoji...")
	emojis, err := s.GetEmoji()
	if err != nil {
		log.Fatalf("failed to get emoji list: %s", err)
	}

	var (
		input   = make(chan emojiData)
		images  = sync.Map{}
		aliases = sync.Map{}
		wg      = sync.WaitGroup{}
	)

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker(input, &images, &aliases, &wg)
	}

	i := 1
	for name, url := range emojis {
		log.Printf("[%d/%d] %s", i, len(emojis), name)
		input <- emojiData{name: name, url: url}
		i += 1
	}

	close(input)
	wg.Wait()

	log.Println("creating aliases...")
	aliases.Range(func(key, value interface{}) bool {
		original, ok := images.Load(value.(string))
		if !ok {
			log.Printf("cannot find existing emoji to alias: %s", value.(string))
			return true
		}
		// log.Printf("aliasing %s -> %s", key.(string), original.(string))
		createAlias(key.(string), original.(string))
		return true
	})

	log.Println("done")
}

func worker(input <-chan emojiData, images, aliases *sync.Map, wg *sync.WaitGroup) {
	for def := range input {
		if strings.HasPrefix(def.url, "alias:") {
			log.Printf("found alias: %s -> %s", def.name, def.url[6:])
			aliases.Store(def.name, def.url[6:])
		} else {
			filename := def.name + filepath.Ext(def.url)
			images.Store(def.name, filename)
			download(filename, def.url)
		}
	}
	log.Println("worker done")
	wg.Done()
}

func download(filename, url string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("failed to create file %s: %s", filename, err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close file %s: %s", filename, err)
		}
	}()

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("failed to GET %s: %s", url, err)
		return
	}
	defer resp.Body.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		log.Printf("failed to write file %s: %s", filename, err)
	}
}

func createAlias(alias, file string) {
	destname := alias + filepath.Ext(file)
	dest, err := os.Create(destname)
	if err != nil {
		log.Printf("failed to create file %s: %s", destname, err)
		return
	}
	defer func() {
		if err := dest.Close(); err != nil {
			log.Printf("failed to close file %s: %s", destname, err)
		}
	}()

	src, err := os.Open(file)
	if err != nil {
		log.Printf("failed to open file %s: %s", file, err)
		return
	}
	defer src.Close()

	if _, err := io.Copy(dest, src); err != nil {
		log.Printf("failed to write file %s: %s", destname, err)
	}
}
