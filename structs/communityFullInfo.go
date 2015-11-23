package structs

type CommunityFullInfo struct {
	CommunityInfo
	Host UserInfo `json:"host"`
}
