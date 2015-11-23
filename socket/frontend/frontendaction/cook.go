package frontendaction

import (
	"encoding/json"
	"hybris/db/dbsession"
	"hybris/enums"
	"hybris/socket/message"

	uppdb "upper.io/db"
)

type CookData struct {
	Ip   string `json:"ip"`
	Auth string `json:"auth"`
}

func Cook(frontend Frontend, msg []byte) (int, interface{}) {
	var data CookData
	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	_, err := dbsession.Get(uppdb.Cond{"cookie": data.Auth})

	return enums.ResponseCodes.Ok, message.S{
		"ok": err == nil,
	}
}
