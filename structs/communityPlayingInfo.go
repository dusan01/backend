package structs

type CommunityPlayingInfo struct {
  DjId    string            `json:"djId"`
  Started string            `json:"started"`
  Media   ResolvedMediaInfo `json:"media"`
  Votes   Votes             `json:"votes"`
}
