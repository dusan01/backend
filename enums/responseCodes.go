package enums

var ResponseCodes = struct {
	Ok,
	BadRequest,
	NotFound,
	Forbidden,
	AlreadyLoggedIn,
	Unimplemented,
	ServerError int
}{
	Ok:              0,
	BadRequest:      1,
	NotFound:        2,
	Forbidden:       3,
	AlreadyLoggedIn: 100,
	Unimplemented:   9998,
	ServerError:     9999,
}
