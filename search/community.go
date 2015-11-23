package search

import (
	"hybris/db/dbcommunity"
	"hybris/realtime"
)

type community struct {
	dbcommunity.Community
	Host              string `json:"host"`
	Population        int    `json:"population"`
	FormatHost        string `json:"-"`
	FormatName        string `json:"-"`
	FormatDescription string `json:"-"`
}

type CommunityResult struct {
	Community Community
	Ranking   int
}

func Community(query string, sortByPopulation bool) {
	var results []CommunityResult
	if len(query) <= 0 {
		for _, c := range realtime.Communities {
			communityData := c.GetCommunity()
			results = apend(results, CommunityResult{
				Community: {
					communityData,
				},
				Ranking: 0,
			})
		}
	}
}
