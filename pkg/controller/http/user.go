package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
)

type UserProfile struct {
	Name string `json:"name"`
}

// userIconHandler handles GET /api/user/{UserID}/icon
func userIconHandler(useCase interfaces.UserUsecases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userID")
		if userID == "" {
			handleError(w, r, goerr.New("user ID is required",
				goerr.T(errs.TagInvalidRequest)))
			return
		}

		imageData, contentType, err := useCase.GetUserIcon(r.Context(), userID)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to get user icon",
				goerr.V("user_id", userID)))
			return
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
		w.WriteHeader(http.StatusOK)

		// Write image data
		if _, err := w.Write(imageData); err != nil {
			// Error already started, can't call handleError
			return
		}
	}
}

// userProfileHandler handles GET /api/user/{UserID}/profile
func userProfileHandler(useCase interfaces.UserUsecases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userID")
		if userID == "" {
			handleError(w, r, goerr.New("user ID is required",
				goerr.T(errs.TagInvalidRequest)))
			return
		}

		name, err := useCase.GetUserProfile(r.Context(), userID)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to get user profile",
				goerr.V("user_id", userID)))
			return
		}

		profile := UserProfile{
			Name: name,
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=600") // Cache for 10 minutes
		w.WriteHeader(http.StatusOK)

		// Write JSON response
		if err := json.NewEncoder(w).Encode(profile); err != nil {
			// Error already started, can't call handleError
			return
		}
	}
}
