package clientaction

import (
	"encoding/json"
	"fmt"
	"hybris/socket/message"
	"time"
)

var actions = map[string]func(Client, []byte) (int, interface{}){
	"adm.broadcast":        AdmBroadcast,
	"adm.globalBan":        AdmGlobalBan,
	"adm.Maintenance":      AdmMaintenance,
	"chat.delete":          ChatDelete,
	"chat.send":            ChatSend,
	"community.create":     CommunityCreate,
	"community.edit":       CommunityEdit,
	"community.getHistory": CommunityGetHistory,
	"community.getInfo":    CommunityGetInfo,
	"community.getStaff":   CommunityGetStaff,
	"community.getState":   CommunityGetState,
	"community.getUsers":   CommunityGetUsers,
	"community.join":       CommunityJoin,
	// "community.search":     CommunitySearch,
	"community.taken": CommunityTaken,
	// "dj.join":               DjJoin,
	// "dj.leave":              DjLeave,
	// "dj.skip":               DjSkip,
	// "media.add":             MediaAdd,
	// "media.import":          MediaImport,
	// "media.search":          MediaSearch,
	// "moderation.addDj":      ModerationAddDj,
	// "moderation.ban":        ModerationBan,
	// "moderation.clearChat":  ModerationClearChat,
	// "moderation.deleteChat": ModerationDeleteChat,
	// "moderation.forceSkip":  ModerationForceSkip,
	// "moderation.kick":       ModerationKick,
	// "moderation.moveDj":     ModerationMoveDj,
	// "moderation.mute":       ModerationMute,
	// "moderation.removeDj":   ModerationRemoveDj,
	// "moderation.setRole":    ModerationSetRole,
	// "playlist.activate":     PlaylistActivate,
	// "playlist.create":       PlaylistCreate,
	// "playlist.delete":       PlaylistDelete,
	// "playlist.edit":         PlaylistEdit,
	// "playlist.get":          PlaylistGet,
	// "playlist.getList":      PlaylistGetList,
	// "playlist.move":         PlaylistMove,
	// "playlistItem.delete":   PlaylistItemDelete,
	// "playlistItem.edit":     PlaylistItemEdit,
	// "playlistItem.move":     PlaylistItemMove,
	// "vote.woot":             VoteWoot,
	// "vote.meh":              VoteMeh,
	// "vote.save":             VoteSave,
	"whoami": Whoami,
}

func Execute(client Client, msg []byte) {
	t := time.Now()

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
		client.Terminate()
		return
	}

	status, data := action(client, frame.Data)
	message.NewAction(frame.Id, status, frame.Action, data).Dispatch(client)
	fmt.Printf("Action %s execution time: %s\n", frame.Action, time.Since(t))
}
