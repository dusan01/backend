package clientaction

import (
	"encoding/json"
	"hybris/db/dbchat"
	"hybris/enums"
	"hybris/socket/message"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func ChatDelete(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Id bson.ObjectId `json:"id"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	client.Lock()
	defer client.Unlock()

	community := client.GetRealtimeUser().GetCommunity()
	if community == nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	chat, err := dbchat.LockGet(data.Id)
	defer dbchat.Unlock(data.Id)
	if err == uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.BadRequest, nil
	} else if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	if chat.UserId != client.GetRealtimeUser().Id {
		return enums.ResponseCodes.Forbidden, nil
	}

	if chat.CommunityId != community.Id || chat.Deleted {
		return enums.ResponseCodes.BadRequest, nil
	}

	chat.Deleted = true
	if err := chat.Save(); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	community.Emit(message.NewEvent("chat.delete", message.S{"id": chat.Id, "deleter": client.GetRealtimeUser().Id}))

	return enums.ResponseCodes.Ok, nil
}
