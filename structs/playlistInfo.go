package structs

type PlaylistInfo struct {
  Name     string `json:"name"`
  Id       string `json:"id"`
  OwnerId  string `json:"ownerId"`
  Selected bool   `json:"selected"`
  Order    int    `json:"order"`

  Length int `json:"length"`
}
