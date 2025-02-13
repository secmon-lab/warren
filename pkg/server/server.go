package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/warren/pkg/usecase"
)

type Server struct {
	router *chi.Mux
}

func New(uc *usecase.UseCases) *Server {
	r := chi.NewRouter()

	r.Use(loggingMiddleware)

	r.Route("/alert", func(r chi.Router) {
		r.Post("/raw/{schema}", alertRawHandler(uc))
		r.Route("/pubsub", func(r chi.Router) {
			r.Use(validateGoogleIDToken)
			r.Post("/{schema}", alertPubSubHandler(uc))
		})
	})
	r.Route("/slack", func(r chi.Router) {
		r.Post("/event", slackEventHandler(uc))
		r.Post("/interaction", slackInteractionHandler(uc))
	})

	return &Server{
		router: r,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
