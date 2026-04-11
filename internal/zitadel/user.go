package zitadel

// UserHumanAdded is the payload of the user.human.added Zitadel event.
//
// See https://zitadel.com/docs/guides/integrate/actions/testing-event
type UserHumanAdded struct {
	UserName          string `json:"userName"`
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	NickName          string `json:"nickName"`
	DisplayName       string `json:"displayName"`
	PreferredLanguage string `json:"preferredLanguage"`
	Gender            int    `json:"gender"`
	Email             string `json:"email"`
}

// UserHumanProfileChanged is the payload of the user.human.profile.changed
// Zitadel event. Only the fields that actually changed are present in the
// JSON body, which is why the mapper rebuilds a FieldMask from the raw
// request bytes via JSON key presence. NickName is a *string because a
// JSON null means "cleared"; the other fields rely on raw-body key
// detection because Go value types cannot distinguish absent from zero.
type UserHumanProfileChanged struct {
	FirstName         string  `json:"firstName"`
	LastName          string  `json:"lastName"`
	NickName          *string `json:"nickName"`
	DisplayName       string  `json:"displayName"`
	PreferredLanguage string  `json:"preferredLanguage"`
	Gender            int     `json:"gender"`
}

// UserHumanEmailChanged is the payload of the user.human.email.changed Zitadel event.
type UserHumanEmailChanged struct {
	Email string `json:"email"`
}

// UserHumanEmailVerified is the payload of the user.human.email.verified Zitadel event.
// It is a marker event with no fields.
type UserHumanEmailVerified struct{}
