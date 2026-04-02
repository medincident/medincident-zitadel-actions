package events

// UserHumanAdded is the payload of the user.human.added Zitadel event.
// See: https://zitadel.com/docs/guides/integrate/actions/testing-event#example-call
type UserHumanAdded struct {
	UserName          string `json:"userName"`
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	NickName          string `json:"nickName"`
	DisplayName       string `json:"displayName"`
	PreferredLanguage string `json:"preferredLanguage"`
	Gender            int    `json:"gender"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
}

// UserHumanProfileChanged is the payload of the user.human.profile.changed Zitadel event.
type UserHumanProfileChanged struct {
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	NickName          string `json:"nickName"`
	DisplayName       string `json:"displayName"`
	PreferredLanguage string `json:"preferredLanguage"`
	Gender            int    `json:"gender"`
}
