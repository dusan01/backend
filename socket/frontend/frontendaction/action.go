package frontendaction

import (
	"encoding/json"
	"hybris/socket/message"
)

var actions = map[string]func(Frontend, []byte) (int, interface{}){
	"cook": Cook,
}

func Execute(frontend Frontend, msg []byte) {
	var frame struct {
		Id     string          `json:"i"`
		Action string          `json:"a"`
		Data   json.RawMessage `json:"d"`
	}

	if err := json.Unmarshal(msg, &frame); err != nil {
		// Handle appropriately
		return
	}

	action, ok := actions[frame.Action]
	if !ok {
		frontend.Terminate()
	}

	status, data := action(frontend, frame.Data)
	message.NewAction(frame.Id, status, frame.Action, data).Dispatch(frontend)
}
