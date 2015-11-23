package enums

var BroadCastTypes = struct {
	SystemAlert,
	SystemAnnouncement,
	GlobalChat,
	Advertisement int
}{
	SystemAlert:        0,
	SystemAnnouncement: 1,
	GlobalChat:         2,
	Advertisement:      3,
}
