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
		w.Write(iconData)
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

		response := map[string]string{
			"profile": profile,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
