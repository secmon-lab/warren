package http

import (
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
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

type Server struct {
	router         *chi.Mux
	slackCtrl      *slack_controller.Controller
	verifier       slack_model.PayloadVerifier
	repo           interfaces.Repository // for GraphQL
	authUC         AuthUseCase           // for authentication
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

func WithAuthUseCase(authUC AuthUseCase) Options {
	return func(s *Server) {
		s.authUC = authUC
	}
}

type UseCase interface {
	interfaces.AlertUsecases
	interfaces.SlackEventUsecases
	interfaces.SlackInteractionUsecases
	interfaces.UserUsecases
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
		graphqlHandler := graphqlHandler(s.repo)

		r.Route("/graphql", func(r chi.Router) {
			// Apply authentication middleware to GraphQL
			if s.authUC != nil {
				r.Use(authMiddleware(s.authUC))
			}

			r.Handle("/", graphqlHandler)
		})

		// Add playground endpoint when GraphiQL is enabled
		if s.enableGraphiQL {
			r.Route("/graphiql", func(r chi.Router) {
				if s.authUC != nil {
					r.Use(authMiddleware(s.authUC))
				}
				r.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
			})
		}
	}

	// Authentication endpoints
	if s.authUC != nil {
		r.Route("/api/auth", func(r chi.Router) {
			r.Get("/login", authLoginHandler(s.authUC))
			r.Get("/callback", authCallbackHandler(s.authUC))
			r.Post("/logout", authLogoutHandler(s.authUC))
			r.Get("/me", authMeHandler(s.authUC))
		})
	}

	// User API endpoints
	r.Route("/api/user", func(r chi.Router) {
		// Apply authentication middleware to user API
		if s.authUC != nil {
			r.Use(authMiddleware(s.authUC))
		}
		r.Get("/{userID}/icon", userIconHandler(uc))
		r.Get("/{userID}/profile", userProfileHandler(uc))
	})

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
				safe.Copy(r.Context(), w, indexFile)
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
