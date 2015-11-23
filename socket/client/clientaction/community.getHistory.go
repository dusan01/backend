package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/db/dbcommunityhistory"
	"hybris/enums"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func CommunityGetHistory(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Id bson.ObjectId `json:"id"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	if _, err := dbcommunity.GetId(data.Id); err == uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.BadRequest, nil
	} else if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	history, err := dbcommunityhistory.GetMulti(50, uppdb.Cond{"communityId": data.Id})
	if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	return enums.ResponseCodes.Ok, dbcommunityhistory.StructMulti(history)
}
