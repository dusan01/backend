package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/enums"
	"hybris/realtime"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func CommunityGetUsers(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Id bson.ObjectId `json:"id"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	communityData, err := dbcommunity.GetId(data.Id)
	if err == uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.BadRequest, nil
	} else if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	community := realtime.NewCommunity(communityData.Id)

	return enums.ResponseCodes.Ok, community.Population
}
