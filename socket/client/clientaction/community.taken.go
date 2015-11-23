package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/enums"
	"hybris/socket/message"

	uppdb "upper.io/db"
)

func CommunityTaken(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Url string `json:"url"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	client.Lock()
	defer client.Unlock()

	_, err := dbcommunity.Get(uppdb.Cond{"url": data.Url})
	return enums.ResponseCodes.Ok, message.S{"taken": err == nil}
}
