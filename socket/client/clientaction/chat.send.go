package clientaction

import (
	"encoding/json"
	"hybris/db/dbchat"
	"hybris/db/dbmute"
	"hybris/enums"
	"hybris/socket/message"
	"time"

	uppdb "upper.io/db"
)

func ChatSend(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Me      bool   `json:"me"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	community := client.GetRealtimeUser().GetCommunity()
	if community == nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	if mute, err := dbmute.Get(uppdb.Cond{"muteeId": client.GetRealtimeUser().Id, "communityId": community.Id}); err == nil {
		if mute.Until == nil || mute.Until.After(time.Now()) {
			return enums.ResponseCodes.Forbidden, nil
		} else if err := mute.Delete(); err != nil {
			return enums.ResponseCodes.ServerError, nil
		}
	} else if err != uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.ServerError, nil
	}

	chat, err := dbchat.New(client.GetRealtimeUser().Id, community.Id, data.Me, data.Message)
	if err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	if err := chat.Save(); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	community.Emit(message.NewEvent("chat.receive", chat.Struct()))

	return enums.ResponseCodes.Ok, nil
}
