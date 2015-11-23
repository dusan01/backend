package validation

import (
	"regexp"
	"strings"
)

var ReservedEmails = []string{
	"turn.dj",
	"turn.fm",
	"ivan.moe",
	"sq10.net",
	"6f.io",
}

func Email(email string) (valid bool) {
	for _, v := range ReservedEmails {
		if strings.HasSuffix(email, v) {
			return
		}
	}

	valid = !(len(email) > 100 ||
		!regexp.MustCompile(`@`).MatchString(email))
	return
}

var ReservedUsernames = []string{
	"ivan", "str_t", "ivanmoe", "ivan.moe", "ivan-moe", "ivan_moe", "str-t", "strt", "str.t",
	"thomas", "6f", "6f7262", "6fio", "6f.io", "6f-io", "6f_io",
	"adis", "funkybasic", "funky.basic", "funky_basic", "funky-basic", "funky", "basic", "adisbasic",
	"sq10", "sq10net", "sq10.net", "sq10_net", "sq10-net",
	"turn",
	"turnfm", "turn.fm", "turnf-fm", "turn_fm",
	"turndj", "turn.dj", "turn-dj", "turn_dj",
	"adm", "admin", "administrator", "administration",
	"about", "terms", "faq", "api", "support", "help", "blog", "chat",
	"u", "r", "_", "me",
	"anonymous", "guest",
	"bot",
	"server",
	"announcement",
}

func Username(username string) (valid bool) {
	for _, v := range ReservedUsernames {
		if username == v {
			return
		}
	}

	valid = regexp.MustCompile(`^[a-z0-9._-]{2,20}$`).MatchString(username)
	return
}

func DisplayName(displayName string) (valid bool) {
	valid = regexp.MustCompile(`^[A-Za-z0-9._-]{2,20}$`).MatchString(displayName)
	return
}

func Password(password string) (valid bool) {
	valid = regexp.MustCompile(`^.{2,72}$`).MatchString(password)
	return
}

func CommunityUrl(communityUrl string) (valid bool) {
	valid = regexp.MustCompile(`^[a-z0-9._-]{2,20}$`).MatchString(communityUrl)
	return
}

func CommunityName(communityName string) (valid bool) {
	length := len(communityName)
	valid = !(length < 2 ||
		length > 30)
	return
}

func CommunityDescription(communityDescription string) (valid bool) {
	valid = !(len(communityDescription) > 500)
	return
}

func CommunityWelcomeMessage(welcomeMessage string) (valid bool) {
	valid = !(len(welcomeMessage) > 300)
	return
}

func Reason(reason string) (valid bool) {
	valid = !(len(reason) > 500)
	return
}

func PlaylistName(playlistName string) (valid bool) {
	length := len(playlistName)
	valid = !(length < 2 ||
		length > 30)
	return
}
