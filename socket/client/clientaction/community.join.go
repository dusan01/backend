package clientaction

import (
	"encoding/json"
	"hybris/db/dbban"
	"hybris/db/dbcommunity"
	"hybris/enums"
	"hybris/realtime"
	"time"

	uppdb "upper.io/db"
)

func CommunityJoin(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Url string `json:"url"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	client.Lock()
	defer client.Unlock()

	communityData, err := dbcommunity.Get(uppdb.Cond{"url": data.Url})
	if err == uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.BadRequest, nil
	} else if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	community := realtime.NewCommunity(communityData.Id)

	if ban, err := dbban.Get(uppdb.Cond{"banneeId": client.GetRealtimeUser().Id, "communityId": community.Id}); err == nil {
		if ban.Until == nil || ban.Until.After(time.Now()) {
			return enums.ResponseCodes.Forbidden, nil
		} else if err := ban.Delete(); err != nil {
			return enums.ResponseCodes.ServerError, nil
		}
	} else if err != uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.ServerError, nil
	}

	// Join community
	community.Join(client.GetRealtimeUser().Id)
	client.GetRealtimeUser().CommunityId = community.Id
	return enums.ResponseCodes.Ok, communityData.Struct()
}
