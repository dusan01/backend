package structs

import ()

type CommunityState struct {
  Waitlist   []string              `json:"waitlist"`
  NowPlaying *CommunityPlayingInfo `json:"nowPlaying"`
}
