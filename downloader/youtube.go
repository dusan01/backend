package downloader

import (
  "code.google.com/p/google-api-go-client/googleapi/transport"
  "code.google.com/p/google-api-go-client/youtube/v3"
  "errors"
  "net/http"
  "strings"
  "time"
)

var ytService *youtube.Service

func init() {
  client := &http.Client{
    Transport: &transport.APIKey{Key: "AIzaSyBAdDIgUc_loht-bJyBtaRcD8aDeupAaeE"},
  }
  var err error
  ytService, err = youtube.New(client)
  if err != nil {
    panic(err)
  }
}

func Youtube(id string) (string, string, string, string, int, error) {
  videoCall := ytService.Videos.List("snippet,contentDetails").Id(id)
  videoResponse, err := videoCall.Do()
  if err != nil {
    return "", "", "", "", 0, err
  }

  if len(videoResponse.Items) <= 0 {
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

  image = "https://i1.ytimg.com/vi/" + id + "/hqdefault.jpg"
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
    blurb = blurb[:396] + "..."
  }

  dur, err := time.ParseDuration(strings.ToLower(item.ContentDetails.Duration[2:]))
  if err != nil {
    return "", "", "", "", 0, err
  }
  length = int(dur.Seconds())

  return image, artist, title, blurb, length, nil
}
