package structs

type HistoryItem struct {
  Dj    string            `json:"dj"`
  Media ResolvedMediaInfo `json:"media"`
  Votes VoteCount         `json:"votes"`
}
