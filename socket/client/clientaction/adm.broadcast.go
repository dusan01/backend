package clientaction

import (
	"encoding/json"
	"hybris/db/dbuser"
	"hybris/enums"
	"hybris/realtime"
	"hybris/socket/message"
	"sync"
)

func AdmBroadcast(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Type    int    `json:"type"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
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

	var wg sync.WaitGroup
	wg.Add(len(realtime.Users))
	evt := message.NewEvent("server.broadcast", data)
	for _, realtimeUser := range realtime.Users {
		go func(realtimeUser *realtime.User) {
			defer wg.Done()
			evt.Dispatch(realtimeUser.Client)
		}(realtimeUser)
	}

	wg.Wait()

	return enums.ResponseCodes.Ok, nil
}
