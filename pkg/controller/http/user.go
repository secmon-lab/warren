package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type UserProfile struct {
	Name string `json:"name"`
}

// userIconHandler returns the user's icon image
func userIconHandler(uc interfaces.ApiUsecases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := chi.URLParam(r, "userID")

		iconData, contentType, err := uc.GetUserIcon(ctx, userID)
		if err != nil {
			http.Error(w, "Failed to get user icon", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
		if _, err := w.Write(iconData); err != nil {
			// Log error but don't try to write error response since headers are already sent
			return
		}
	}
}

// userProfileHandler returns the user's profile information
func userProfileHandler(uc interfaces.ApiUsecases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := chi.URLParam(r, "userID")

		profile, err := uc.GetUserProfile(ctx, userID)
		if err != nil {
			http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
			return
		}

		response := UserProfile{
			Name: profile,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log error but don't try to write error response since headers are already sent
			return
		}
	}
}
