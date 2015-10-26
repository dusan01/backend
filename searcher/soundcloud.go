package searcher

import (
	"encoding/json"
	"fmt"
	"hybris/structs"
	"net/http"
	"net/url"
	"strings"
)

func SearchSoundcloud(query string) ([]structs.SearchResult, error) {
	results := []structs.SearchResult{}
	var out []struct {
		Image string `json:"artwork_url"`
		Id    int    `json:"id"`
		Title string `json:"title"`
		User  struct {
			Username string `json:"username"`
		} `json:"user"`
	}

	res, err := http.Get("https://api.soundcloud.com/tracks?client_id=fddfcd9f79c36f4716b4f7ab1664cd8d&limit=50&q=" +
		url.QueryEscape(query))
	if err != nil {
		return results, err
	}

	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return results, err
	}

	for _, item := range out {
		var (
			artist string
			title  string = item.Title
		)

		strSplit := strings.Split(title, " - ")
		if len(strSplit) > 1 {
			artist = strSplit[0]
			title = strings.Join(strSplit[1:], " - ")
		} else {
			artist = item.User.Username
		}

		searchResult := structs.SearchResult{
			Image:   item.Image,
			Artist:  artist,
			Title:   title,
			Type:    1,
			MediaId: fmt.Sprintf("%d", item.Id),
		}
		results = append(results, searchResult)
	}

	return results, nil
}
