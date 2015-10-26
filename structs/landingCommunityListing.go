package structs

type LandingCommunityListing struct {
  Population int                      `json:"population"`
  Playing    CommunityFullPlayingInfo `json:"playing"`
  Info       CommunityFullInfo        `json:"info"`
}
