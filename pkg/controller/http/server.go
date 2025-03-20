package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	slack_controller "github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
)

type Server struct {
	router    *chi.Mux
	slackCtrl *slack_controller.Controller
	verifier  slack_model.PayloadVerifier
}

type Options func(*Server)

func WithSlackVerifier(verifier slack_model.PayloadVerifier) Options {
	return func(s *Server) {
		s.verifier = verifier
	}
}

func New(uc interfaces.UseCase, opts ...Options) *Server {
	r := chi.NewRouter()

	s := &Server{
		router:    r,
		slackCtrl: slack_controller.New(uc),
	}
	for _, opt := range opts {
		opt(s)
	}

	r.Use(loggingMiddleware)
	r.Use(panicRecoveryMiddleware)

	r.Route("/alert", func(r chi.Router) {
		r.Use(withAuthHTTPRequest)

		r.Post("/raw/{schema}", alertRawHandler(uc))
		r.Route("/pubsub", func(r chi.Router) {
			r.Use(validateGoogleIDToken)
			r.Post("/{schema}", alertPubSubHandler(uc))
		})
		r.Route("/sns", func(r chi.Router) {
			r.Use(verifySNSRequest)
			r.Post("/{schema}", alertSNSHandler(uc))
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
