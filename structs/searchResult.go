package structs

type SearchResult struct {
	Image   string `json:"img"`
	Artist  string `json:"artist"`
	Title   string `json:"title"`
	Type    int    `json:"type"`
	MediaId string `json:"mid"`
}
