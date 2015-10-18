package searcher

import (
  "code.google.com/p/google-api-go-client/googleapi/transport"
  "code.google.com/p/google-api-go-client/youtube/v3"
  "hybris/structs"
  "net/http"
  "strings"
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

func SearchYoutube(query string) ([]structs.SearchResult, error) {
  results := []structs.SearchResult{}
  searchCall := ytService.Search.List("id,snippet").
    Q(query).
    Type("video").
    MaxResults(50)
  searchResponse, err := searchCall.Do()
  if err != nil {
    return results, err
  }

  for _, item := range searchResponse.Items {
    var (
      artist string
      title  string = item.Snippet.Title
    )

    strSplit := strings.Split(title, " - ")
    if len(strSplit) > 1 {
      artist = strSplit[0]
      title = strings.Join(strSplit[1:], " - ")
    } else {
      artist = item.Snippet.ChannelTitle
    }

    searchResult := structs.SearchResult{
      Image:   "https://i1.ytimg.com/vi/" + item.Id.VideoId + "/hqdefault.jpg",
      Artist:  artist,
      Title:   title,
      Type:    0,
      MediaId: item.Id.VideoId,
    }
    results = append(results, searchResult)
  }

  return results, nil
}
