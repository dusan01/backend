package structs

import ()

type CommunityPlayingInfo struct {
  DjId  string            `json:"djId"`
  Media ResolvedMediaInfo `json:"media"`
  Votes Votes             `json:"votes"`
}
