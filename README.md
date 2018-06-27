# Slack Emoji Downloader

Using this script, you can download your Slack team's entire emoji set to disk!
To run it, you'll need:

1. A recent version of Firefox
2. Go

## To Run It

First, start Firefox in
[Marionette](https://firefox-source-docs.mozilla.org/testing/marionette/marionette/index.html)
mode. When you do, the address bar should turn orange and you should see a robot head next
to the connection status icon. Then run

```bash
$ go get github.com/njasm/marionette_client
$ go run download.go -team <team> -o <directory>
```

where `<team>` is the name of your Slack organization as it appears in your
URLs, and `<directory>` the directory you want to save all the images to.  The
script will then use Firefox to crawl the Slack customization pages, downloading
every Emoji in its path.
