package structs

import ()

type HistoryItem struct {
  Dj    string            `json:"dj"`
  Media ResolvedMediaInfo `json:"media"`
  Votes Votes             `json:"votes"`
}
