# Slack Emoji Downloader

Using this script, you can download your Slack team's entire emoji set to disk!
To run it, you'll need a Slack API token. You can follow a process [like
this](https://github.com/jackellenberger/emojme#finding-a-slack-token) to get
one.

## To Run It

```bash
$ go run download.go -output <directory> -token <api_token>
```

# Slack Emoji Uploader

I need to figure out of the uploader still works, but at least the downloader
is fixed.
