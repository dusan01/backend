package structs

type CommunityState struct {
  Waitlist   []string              `json:"waitlist"`
  NowPlaying *CommunityPlayingInfo `json:"nowPlaying"`
}
