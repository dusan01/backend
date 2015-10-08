package structs

import ()

type PlaylistItem struct {
  PlaylistId string `json:"playlistId"`
  Title      string `json:"title"`
  Artist     string `json:"artist"`
  Order      int    `json:"order"`

  Media MediaInfo `json:"media"`
}
