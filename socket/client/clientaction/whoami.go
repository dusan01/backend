package clientaction

import (
	"hybris/db/dbuser"
	"hybris/enums"
)

func Whoami(client Client, msg []byte) (int, interface{}) {
	user, err := dbuser.GetId(client.GetRealtimeUser().Id)
	if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}
	return enums.ResponseCodes.Ok, user
}
