package clientaction

import (
	"encoding/json"
	"hybris/db/dbuser"
	"hybris/enums"
	"hybris/realtime"
	"sync"
)

func AdmMaintenance(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Start bool `json:"start"`
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

	realtime.Maintenance = data.Start

	if realtime.Maintenance {
		var wg sync.WaitGroup

		wg.Add(len(realtime.Users))
		for _, realtimeUser := range realtime.Users {
			go func(realtimeUser *realtime.User) {
				defer wg.Done()
				u, err := dbuser.GetId(realtimeUser.Id)
				if err != nil {
					return
				}

				if u.GlobalRole < enums.GlobalRoles.Admin {
					realtimeUser.Panic()
				}
			}(realtimeUser)
		}

		wg.Wait()
	}

	return enums.ResponseCodes.Ok, nil
}
