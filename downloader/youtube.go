package downloader

import (
	"errors"
	"hybris/debug"
	"net/http"
	"strings"
	"time"

	"code.google.com/p/google-api-go-client/googleapi/transport"
	youtube "code.google.com/p/google-api-go-client/youtube/v3"
)

var ytService *youtube.Service

func init() {
	debug.Log("Creating youtube oAuth service")
	client := &http.Client{
		Transport: &transport.APIKey{Key: "AIzaSyBAdDIgUc_loht-bJyBtaRcD8aDeupAaeE"},
	}
	var err error
	ytService, err = youtube.New(client)
	if err != nil {
		debug.Log("Failed to create youtube oAuth service: %s", err.Error())
		panic(err)
	}
}

func Youtube(id string) (string, string, string, string, int, error) {
	debug.Log("Downloading media info for %s from youtube", id)
	videoCall := ytService.Videos.List("snippet,contentDetails").
		Id(id)
	videoResponse, err := videoCall.Do()
	if err != nil {
		debug.Log("Failed to download media for %s info from youtube: %s", id, err.Error())
		return "", "", "", "", 0, err
	}

	if len(videoResponse.Items) <= 0 {
		debug.Log("Youtube returned 0 results on query for %s", id)
		return "", "", "", "", 0, errors.New("Youtube API returned no media")
	}

	item := videoResponse.Items[0]

	var (
		image  string
		artist string
		title  string
		blurb  string
		length int
	)

	image = "https://img.youtube.com/vi/" + id + "/hqdefault.jpg"
	title = item.Snippet.Title
	blurb = item.Snippet.Description

	strSplit := strings.Split(title, " - ")
	if len(strSplit) > 1 {
		artist = strSplit[0]
		title = strings.Join(strSplit[1:], " - ")
	} else {
		artist = item.Snippet.ChannelTitle
	}

	if len(blurb) > 400 {
		blurb = blurb[:397] + "..."
	}

	dur, err := time.ParseDuration(strings.ToLower(item.ContentDetails.Duration[2:]))
	if err != nil {
		debug.Log("Could not parse media duration for %s from youtube: %s", id, err.Error())
		return "", "", "", "", 0, err
	}
	length = int(dur.Seconds())

	debug.Log("Successfully downloaded media info for %s from youtube", id)
	return image, artist, title, blurb, length, nil
}
