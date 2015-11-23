package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/enums"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func CommunityGetInfo(client Client, msg []byte) (int, interface{}) {
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

	return enums.ResponseCodes.Ok, communityData.Struct()
}
