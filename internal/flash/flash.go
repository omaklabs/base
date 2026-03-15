// Package flash provides cookie-based flash messages.
// Set() stores a message, middleware.FlashContext reads it into the
// view context, and templates render it via components.FlashMessage().
package flash

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
)

// Flash holds a single message to display to the user on the next page load.
type Flash struct {
	Message string `json:"message"`
	Variant string `json:"variant"` // success, error, warning, info
}

const cookieName = "flash"

// Set encodes a flash message and stores it in a short-lived cookie. The cookie
// is HttpOnly and scoped to the root path so it is available on the next
// request regardless of the URL.
func Set(w http.ResponseWriter, message, variant string) {
	f := Flash{Message: message, Variant: variant}
	data, _ := json.Marshal(f)
	encoded := base64.URLEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
	})
}

// Get reads the flash cookie from the request and decodes it. Returns nil if
// no flash cookie is present or the value cannot be decoded.
func Get(r *http.Request) *Flash {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	data, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}

	var f Flash
	if err := json.Unmarshal(data, &f); err != nil {
		return nil
	}

	return &f
}

// Clear removes the flash cookie by setting its MaxAge to -1.
func Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// GetAndClear reads the flash cookie and immediately clears it so the message
// is only shown once. This is the function handlers should use in most cases.
func GetAndClear(w http.ResponseWriter, r *http.Request) *Flash {
	f := Get(r)
	if f != nil {
		Clear(w)
	}
	return f
}
