package zitadel

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
}

// UserHumanProfileChanged is the payload of the user.human.profile.changed Zitadel event.
// For FieldMask building we use raw JSON key presence detection (see mapper).
// NickName is a pointer because JSON null means "cleared"; value types use
// presence detection via the raw body instead.
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
