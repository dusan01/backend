package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/db/dbcommunitystaff"
	"hybris/enums"
	"hybris/realtime"

	uppdb "upper.io/db"
)

func CommunityCreate(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Url  string `json:"url"`
		Name string `json:"name"`
		Nsfw bool   `json:"nsfw"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	client.Lock()
	defer client.Unlock()

	communities, err := dbcommunity.GetMulti(-1, uppdb.Cond{"hostId": client.GetRealtimeUser().Id})
	if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	if len(communities) >= 3 {
		return enums.ResponseCodes.Forbidden, nil
	}

	community, err := dbcommunity.New(client.GetRealtimeUser().Id, data.Url, data.Name, data.Nsfw)
	if err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	if err := community.Save(); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	staffObject, err := dbcommunitystaff.New(community.Id, client.GetRealtimeUser().Id, enums.ModerationRoles.Host)
	if err := staffObject.Save(); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	realtime.NewCommunity(community.Id)

	return enums.ResponseCodes.Ok, nil
}
