package downloader

import (
	"encoding/json"
	"errors"
	"hybris/debug"
	"net/http"
	"strings"
)

func Soundcloud(id string) (string, string, string, string, int, error) {
	debug.Log("Downloading media info for %s from soundcloud", id)
	var out struct {
		Image       string `json:"artwork_url"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Length      int    `json:"duration"`
		User        struct {
			Username string `json:"username"`
		} `json:"user"`
	}

	res, err := http.Get("https://api.soundcloud.com/tracks/" + id + "?client_id=fddfcd9f79c36f4716b4f7ab1664cd8d")
	if err != nil {
		debug.Log("Failed to retrieve media info for %s from soundcloud: %s", id, err.Error())
		return "", "", "", "", 0, err
	}

	if res.StatusCode != 200 {
		debug.Log("Failed to retrieve media info for %s from soundcloud: Expected response code 200, received %d",
			res.StatusCode)
		return "", "", "", "", 0, errors.New("Failed to get media")
	}

	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		debug.Log("Failed to unmarshal json response for %s from soundcloud: %s", err.Error())
		return "", "", "", "", 0, err
	}

	var (
		image  string
		artist string
		title  string
		blurb  string
		length int
	)

	image = out.Image
	title = out.Title
	blurb = out.Description
	length = out.Length / 1000

	strSplit := strings.Split(title, " - ")
	if len(strSplit) > 1 {
		artist = strSplit[0]
		title = strings.Join(strSplit[1:], " - ")
	} else {
		artist = out.User.Username
	}

	if len(blurb) > 400 {
		blurb = blurb[:397] + "..."
	}

	debug.Log("Successfully downloaded media info for %s from soundcloud", id)
	return image, artist, title, blurb, length, nil
}
