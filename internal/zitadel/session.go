package zitadel

// SessionAdded is the payload of the session.added Zitadel event.
type SessionAdded struct {
	UserAgent *SessionUserAgent `json:"user_agent,omitempty"`
}

// SessionUserAgent contains device/browser info from the Zitadel session.
type SessionUserAgent struct {
	FingerprintID *string `json:"fingerprint_id,omitempty"`
	IP            string  `json:"ip,omitempty"`
	Description   *string `json:"description,omitempty"`
}

// SessionUserChecked is the payload of the session.user.checked Zitadel event.
type SessionUserChecked struct {
	UserID            string `json:"userID"`
	UserResourceOwner string `json:"userResourceOwner"`
}
