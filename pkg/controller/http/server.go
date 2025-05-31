package http

import (
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/warren/frontend"
	"github.com/secmon-lab/warren/pkg/controller/graphql"
	slack_controller "github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
)

type Server struct {
	router         *chi.Mux
	slackCtrl      *slack_controller.Controller
	verifier       slack_model.PayloadVerifier
	repo           interfaces.Repository // for GraphQL
	enableGraphiQL bool                  // GraphiQL enable flag
}

type Options func(*Server)

func WithSlackVerifier(verifier slack_model.PayloadVerifier) Options {
	return func(s *Server) {
		s.verifier = verifier
	}
}

func WithGraphQLRepo(repo interfaces.Repository) Options {
	return func(s *Server) {
		s.repo = repo
	}
}

func WithGraphiQL(enabled bool) Options {
	return func(s *Server) {
		s.enableGraphiQL = enabled
	}
}

type UseCase interface {
	interfaces.AlertUsecases
	interfaces.SlackEventUsecases
	interfaces.SlackInteractionUsecases
}

func New(uc UseCase, opts ...Options) *Server {
	r := chi.NewRouter()

	s := &Server{
		router:    r,
		slackCtrl: slack_controller.New(uc, uc),
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

	// GraphQL endpoint
	if s.repo != nil {
		r.Handle("/graphql", graphqlHandler(s.repo))

		// Add playground endpoint when GraphiQL is enabled
		if s.enableGraphiQL {
			r.Handle("/graphiql", playground.Handler("GraphQL playground", "/graphql"))
		}
	}

	// Static file serving for SPA
	staticFS, err := fs.Sub(frontend.StaticFiles, "dist")
	if err == nil {
		// Serve static files and handle SPA routing
		r.HandleFunc("/*", spaHandler(staticFS))
	}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GraphQL handler
func graphqlHandler(repo interfaces.Repository) http.Handler {
	resolver := graphql.NewResolver(repo)
	srv := handler.New(
		graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}),
	)
	srv.AddTransport(transport.POST{})
	return srv
}

// spaHandler handles SPA routing by serving static files and falling back to index.html
func spaHandler(staticFS fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(staticFS))

	return func(w http.ResponseWriter, r *http.Request) {
		urlPath := strings.TrimPrefix(r.URL.Path, "/")

		// If the path is empty, serve index.html
		if urlPath == "" {
			urlPath = "index.html"
		}

		// Try to open the file
		if _, err := staticFS.Open(urlPath); err != nil {
			// For SPA routes (not assets), serve index.html for client-side routing
			// Assets like _next/, api/, static/, etc. should return 404
			if strings.HasPrefix(urlPath, "_next/") ||
				strings.HasPrefix(urlPath, "api/") ||
				strings.HasPrefix(urlPath, "static/") ||
				strings.Contains(urlPath, ".") { // Files with extensions
				// File not found, return 404
				http.NotFound(w, r)
				return
			}

			// For SPA routes, serve index.html
			if indexFile, err := staticFS.Open("index.html"); err == nil {
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html")
				io.Copy(w, indexFile)
				return
			}

			// If index.html is also not found, return 404
			http.NotFound(w, r)
			return
		}

		// Serve the requested file
		fileServer.ServeHTTP(w, r)
	}
}
