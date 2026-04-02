package events

// UserHumanAdded is the payload of the user.human.added Zitadel event.
// See: https://zitadel.com/docs/guides/integrate/actions/testing-event#example-call
type UserHumanAdded struct {
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	NickName          string `json:"nickName"`
	DisplayName       string `json:"displayName"`
	UserName          string `json:"userName"`
	Email             string `json:"email"`
	PreferredLanguage string `json:"preferredLanguage"`
}
