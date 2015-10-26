package structs

type MediaInfo struct {
  Id        string `json:"id"`
  Type      int    `json:"type"`
  MediaId   string `json:"mid"`
  Image     string `json:"img"`
  Length    int    `json:"length"`
  Title     string `json:"title"`
  Artist    string `json:"artist"`
  Blurb     string `json:"blurb"`
  Plays     int    `json:"plays"`
  Woots     int    `json:"woots"`
  Mehs      int    `json:"mehs"`
  Saves     int    `json:"saves"`
  Playlists int    `json:"playlists"`
}
