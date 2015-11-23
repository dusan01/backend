package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/db/dbcommunitystaff"
	"hybris/enums"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func CommunityGetStaff(client Client, msg []byte) (int, interface{}) {
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

	history, err := dbcommunitystaff.GetMulti(50, uppdb.Cond{"communityId": data.Id})
	if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	return enums.ResponseCodes.Ok, dbcommunitystaff.StructMulti(history)
}
