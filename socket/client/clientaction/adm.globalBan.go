package clientaction

import (
	"encoding/json"
	"hybris/db/dbglobalban"
	"hybris/db/dbuser"
	"hybris/enums"
	"hybris/realtime"
	"hybris/socket/message"
	"time"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func AdmGlobalBan(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Id       bson.ObjectId `json:"id"`
		Duration time.Duration `json:"duration"`
		Reason   string        `json:"reason"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, err.Error()
	}

	client.Lock()
	defer client.Unlock()

	user, err := dbuser.GetId(client.GetRealtimeUser().Id)
	if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	if user.GlobalRole < enums.GlobalRoles.Admin {
		return enums.ResponseCodes.Forbidden, nil
	}

	bannee, err := dbuser.GetId(data.Id)
	if err == uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.BadRequest, nil
	} else if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	until := time.Now().Add(data.Duration * time.Second)

	globalBan, err := dbglobalban.New(bannee.Id, client.GetRealtimeUser().Id, data.Reason, &until)
	if err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	if err := globalBan.Save(); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	if u, ok := realtime.Users[bannee.Id]; ok {
		community := u.GetCommunity()
		u.Panic()
		if community != nil {
			community.Emit(message.NewEvent("globalBan", message.S{"banner": client.GetRealtimeUser().Id, "bannee": bannee.Id}))
		}
	}

	return enums.ResponseCodes.Ok, nil
}
