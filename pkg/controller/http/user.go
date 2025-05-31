package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
)

type UserIconUseCase interface {
	GetUserIcon(ctx context.Context, userID string) ([]byte, string, error)
}

// userIconHandler handles GET /api/user/{UserID}/icon
func userIconHandler(useCase UserIconUseCase) http.HandlerFunc {
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
