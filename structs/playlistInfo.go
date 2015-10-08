package structs

import ()

type PlaylistInfo struct {
  Name     string `json:"name"`
  Id       string `json:"id"`
  OwnerId  string `json:"ownerId"`
  Selected bool   `json:"selected"`

  Length int `json:"length"`
}
