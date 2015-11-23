package structs

type CommunityFullPlayingInfo struct {
	CommunityPlayingInfo
	Dj UserInfo `json:"dj"`
}
