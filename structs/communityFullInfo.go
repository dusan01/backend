package structs

import ()

type CommunityFullInfo struct {
  CommunityInfo
  Host UserInfo `json:"host"`
}
