package clientaction

import (
	"encoding/json"
	"hybris/db/dbcommunity"
	"hybris/enums"
	"hybris/realtime"
	"hybris/validation"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

func CommunityEdit(client Client, msg []byte) (int, interface{}) {
	var data struct {
		Id              bson.ObjectId `json:"id"`
		Name            *string       `json:"name"`
		Description     *string       `json:"description"`
		WelcomeMessage  *string       `json:"welcomeMessage"`
		WaitlistEnabled *bool         `json:"waitlistEnabled"`
		DjRecycling     *bool         `json:"djRecycling"`
		Nsfw            *bool         `json:"nsfw"`
	}

	if err := json.Unmarshal(msg, &data); err != nil {
		return enums.ResponseCodes.BadRequest, nil
	}

	client.Lock()
	defer client.Unlock()

	communityData, err := dbcommunity.LockGet(data.Id)
	defer dbcommunity.Unlock(data.Id)
	if err == uppdb.ErrNoMoreRows {
		return enums.ResponseCodes.BadRequest, nil
	} else if err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	community := realtime.NewCommunity(communityData.Id)
	if !community.HasPermission(client.GetRealtimeUser().Id, enums.ModerationRoles.Manager) {
		return enums.ResponseCodes.Forbidden, nil
	}

	if data.Name != nil {
		name := *data.Name
		if !validation.CommunityName(name) {
			return enums.ResponseCodes.BadRequest, nil
		}
		communityData.Name = name
	}

	if data.Description != nil {
		description := *data.Description
		if !validation.CommunityDescription(description) {
			return enums.ResponseCodes.BadRequest, nil
		}
		communityData.Description = description
	}

	if data.WelcomeMessage != nil {
		welcomeMessage := *data.WelcomeMessage
		if !validation.CommunityWelcomeMessage(welcomeMessage) {
			return enums.ResponseCodes.BadRequest, nil
		}
		communityData.WelcomeMessage = welcomeMessage
	}

	if data.WaitlistEnabled != nil {
		communityData.WaitlistEnabled = *data.WaitlistEnabled
	}

	if data.DjRecycling != nil {
		communityData.DjRecycling = *data.DjRecycling
	}

	if data.Nsfw != nil {
		communityData.Nsfw = *data.Nsfw
	}

	if err := communityData.Save(); err != nil {
		return enums.ResponseCodes.ServerError, nil
	}

	// Dispatch a global event to notify all users in the community that data has been updated
	return enums.ResponseCodes.Ok, communityData.Struct()
}
