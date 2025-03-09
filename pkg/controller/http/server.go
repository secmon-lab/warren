package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/interfaces"
)

type Server struct {
	router    *chi.Mux
	slackCtrl *slack.Controller
	verifier  interfaces.SlackPayloadVerifier
}

type Options func(*Server)

func WithSlackVerifier(verifier interfaces.SlackPayloadVerifier) Options {
	return func(s *Server) {
		s.verifier = verifier
	}
}

func New(uc interfaces.UseCase, opts ...Options) *Server {
	r := chi.NewRouter()

	s := &Server{
		router:    r,
		slackCtrl: slack.New(uc),
	}
	for _, opt := range opts {
		opt(s)
	}

	r.Use(loggingMiddleware)
	r.Use(panicRecoveryMiddleware)

	r.Route("/alert", func(r chi.Router) {
		r.Post("/raw/{schema}", alertRawHandler(uc))
		r.Route("/pubsub", func(r chi.Router) {
			r.Use(validateGoogleIDToken)
			r.Post("/{schema}", alertPubSubHandler(uc))
		})
	})
	r.Route("/slack", func(r chi.Router) {
		r.Use(verifySlackRequest(s.verifier))
		r.Post("/event", slackEventHandler(s.slackCtrl))
		r.Post("/interaction", slackInteractionHandler(s.slackCtrl))
	})

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
