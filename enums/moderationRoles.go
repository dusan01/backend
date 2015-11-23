package enums

var ModerationRoles = struct {
	Guest,
	User,
	Dj,
	Bouncer,
	Manager,
	CoHost,
	Host int
}{
	Guest:   0,
	User:    1,
	Dj:      2,
	Bouncer: 3,
	Manager: 4,
	CoHost:  5,
	Host:    6,
}
