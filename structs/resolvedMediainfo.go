package structs

import ()

type ResolvedMediaInfo struct {
  MediaInfo
  Artist string `json:"artist"`
  Title  string `json:"title"`
}
