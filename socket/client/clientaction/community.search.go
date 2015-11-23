package clientaction

import (
	"encoding/json"
	"hybris/enums"
)

func CommunitySearch(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Query            string `json:"query"`
		Offset           int    `json:"offset"`
		SortByPopulation bool   `json:"sortByPop"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	return enums.ResponseCodes.Ok, nil
}
