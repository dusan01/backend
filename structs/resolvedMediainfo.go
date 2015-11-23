package structs

type ResolvedMediaInfo struct {
	MediaInfo
	Artist string `json:"artist"`
	Title  string `json:"title"`
}
