package structs

import ()

type CommunityFullPlayingInfo struct {
  CommunityPlayingInfo
  Dj UserInfo `json:"dj"`
}
