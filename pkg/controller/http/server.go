package http

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/warren/frontend"
	"github.com/secmon-lab/warren/pkg/controller/graphql"
	slack_controller "github.com/secmon-lab/warren/pkg/controller/slack"
	websocket_controller "github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// AsyncAlertHookConfig represents configuration for asynchronous alert hooks
type AsyncAlertHookConfig struct {
	Raw    bool
	PubSub bool
	SNS    bool
}

type Server struct {
	router          *chi.Mux
	slackCtrl       *slack_controller.Controller
	websocketCtrl   *websocket_controller.Handler // for WebSocket chat
	policy          interfaces.PolicyClient
	verifier        slack_model.PayloadVerifier
	repo            interfaces.Repository // for GraphQL
	slackService    *slack.Service        // for GraphQL resolver
	authUC          AuthUseCase           // for authentication
	enableGraphiQL  bool                  // GraphiQL enable flag
	noAuthorization bool                  // no-authorization flag
	asyncAlertHook  *AsyncAlertHookConfig // async alert hook configuration
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

func WithSlackService(slackService *slack.Service) Options {
	return func(s *Server) {
		s.slackService = slackService
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

func WithPolicy(policy interfaces.PolicyClient) Options {
	return func(s *Server) {
		s.policy = policy
	}
}

func WithNoAuthorization(disabled bool) Options {
	return func(s *Server) {
		s.noAuthorization = disabled
	}
}

func WithWebSocketHandler(handler *websocket_controller.Handler) Options {
	return func(s *Server) {
		s.websocketCtrl = handler
	}
}

func WithAsyncAlertHook(cfg *AsyncAlertHookConfig) Options {
	return func(s *Server) {
		s.asyncAlertHook = cfg
	}
}

type UseCase interface {
	interfaces.AlertUsecases
	interfaces.SlackEventUsecases
	interfaces.SlackInteractionUsecases
	interfaces.ApiUsecases
}

// Static file extensions that should be served directly (not fallback to SPA)
var staticFileExtensions = []string{
	".ico",            // favicon files
	".png",            // favicon PNG files
	".svg",            // SVG files
	".css",            // CSS files
	".js",             // JavaScript files
	".woff", ".woff2", // Web fonts
	".ttf", ".otf", // Font files
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
	r.Use(withAuthHTTPRequest)
	r.Use(validateGoogleIAPToken)
	r.Use(withAsyncConfig(s.asyncAlertHook))

	// Migration to /hooks
	r.Route("/hooks", func(r chi.Router) {
		r.Route("/alert", func(r chi.Router) {
			r.Route("/raw", func(r chi.Router) {
				r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))
				r.Post("/{schema}", alertRawHandler(uc))
			})
			r.Route("/pubsub", func(r chi.Router) {
				r.Use(validateGoogleIDToken)
				r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))
				r.Post("/{schema}", alertPubSubHandler(uc))
			})
			r.Route("/sns", func(r chi.Router) {
				r.Use(verifySNSRequest)
				r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))
				r.Post("/{schema}", alertSNSHandler(uc))
			})
		})

		r.Route("/slack", func(r chi.Router) {
			r.Use(verifySlackRequest(s.verifier))
			r.Post("/event", slackEventHandler(s.slackCtrl))
			r.Post("/interaction", slackInteractionHandler(s.slackCtrl))
		})
	})

	// GraphQL endpoint
	if s.repo != nil {
		graphqlHandler := graphqlHandler(s.repo, s.slackService, uc)

		r.Route("/graphql", func(r chi.Router) {
			// Apply authentication middleware to GraphQL
			if s.authUC != nil {
				r.Use(authMiddleware(s.authUC))
			}

			r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))
			r.Handle("/", graphqlHandler)
		})

		// Add playground endpoint when GraphiQL is enabled
		if s.enableGraphiQL {
			r.Route("/graphiql", func(r chi.Router) {
				if s.authUC != nil {
					r.Use(authMiddleware(s.authUC))
				}
				r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))
				r.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
			})
		}
	}

	// Authentication endpoints
	if s.authUC != nil {
		r.Route("/api", func(r chi.Router) {
			r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))
			r.Route("/auth", func(r chi.Router) {
				r.Get("/login", authLoginHandler(s.authUC))
				r.Get("/callback", authCallbackHandler(s.authUC))
				r.Post("/logout", authLogoutHandler(s.authUC))
				r.Get("/me", authMeHandler(s.authUC))
			})
			r.Route("/user", func(r chi.Router) {
				r.Use(authMiddleware(s.authUC))
				r.Get("/{userID}/icon", userIconHandler(uc))
				r.Get("/{userID}/profile", userProfileHandler(uc))
			})
			r.Route("/tickets", func(r chi.Router) {
				r.Use(authMiddleware(s.authUC))
				r.Get("/{ticketID}/alerts/download", ticketAlertsDownloadHandler(uc))
			})
		})
	}

	// WebSocket endpoints
	if s.websocketCtrl != nil {
		r.Route("/ws", func(r chi.Router) {
			// Apply authentication middleware to WebSocket endpoints
			r.Use(authMiddleware(s.authUC))
			r.Use(authorizeWithPolicy(s.policy, s.noAuthorization))

			r.Route("/chat", func(r chi.Router) {
				r.Get("/ticket/{ticketID}", s.websocketCtrl.HandleTicketChat)
			})
		})
	}

	// Static file serving for SPA
	staticFS, err := fs.Sub(frontend.StaticFiles, "dist")
	if err == nil {
		// Check if index.html exists
		if _, err := staticFS.Open("index.html"); err == nil {
			// Dedicated favicon handlers for better reliability
			r.Get("/favicon.ico", faviconHandler(staticFS, "favicon.ico", "image/x-icon"))
			r.Get("/favicon-192x192.png", faviconHandler(staticFS, "favicon-192x192.png", "image/png"))
			r.Get("/favicon-512x512.png", faviconHandler(staticFS, "favicon-512x512.png", "image/png"))

			// Serve static files and handle SPA routing
			r.HandleFunc("/*", spaHandler(staticFS))
		}
	}

	return s
}

// ticketAlertsDownloadHandler handles downloading alert data as JSONL for a specific ticket
func ticketAlertsDownloadHandler(uc UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ticketID := chi.URLParam(r, "ticketID")

		// Parse ticket ID
		tID := types.TicketID(ticketID)

		// Generate JSONL data using usecase
		jsonlData, err := uc.GenerateTicketAlertsJSONL(ctx, tID)
		if err != nil {

			// Check if it's a "not found" error
			if strings.Contains(err.Error(), "ticket not found") {
				http.Error(w, "Ticket not found", http.StatusNotFound)
				return
			}

			http.Error(w, "Failed to generate alert data", http.StatusInternalServerError)
			return
		}

		// Set response headers for file download
		filename := fmt.Sprintf("ticket-%s-alerts-%s.jsonl", ticketID, time.Now().Format("20060102-150405"))
		w.Header().Set("Content-Type", "application/jsonl")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

		// Write JSONL content
		if _, err := w.Write(jsonlData); err != nil {
			// Log error but don't try to write error response since headers are already sent
			// This follows the pattern of other handlers in the codebase
			return
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GraphQL handler
func graphqlHandler(repo interfaces.Repository, slackService *slack.Service, uc UseCase) http.Handler {
	var useCases *usecase.UseCases
	if uc != nil {
		// Type assertion to convert interface to concrete type
		var ok bool
		useCases, ok = uc.(*usecase.UseCases)
		if !ok {
			panic("uc must be of type *usecase.UseCases")
		}
	}
	resolver := graphql.NewResolver(repo, slackService, useCases)
	srv := handler.New(
		graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}),
	)
	srv.AddTransport(transport.POST{})

	// Add DataLoader middleware using the official implementation
	var slackClient interfaces.SlackClient
	if slackService != nil {
		slackClient = slackService.GetClient()
	}

	return graphql.DataLoaderMiddleware(repo, slackClient)(srv)
}

// faviconHandler serves favicon files with appropriate Content-Type headers
func faviconHandler(staticFS fs.FS, filename, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := staticFS.Open(filename)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer safe.Close(r.Context(), file)

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		safe.Copy(r.Context(), w, file)
	}
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

		// Try to open the file to check if it exists
		if file, err := staticFS.Open(urlPath); err != nil {
			// File not found

			// For SPA routes (not assets), serve index.html for client-side routing
			// But first check if this looks like an asset request
			isStaticFile := strings.HasPrefix(urlPath, "_next/") ||
				strings.HasPrefix(urlPath, "api/") ||
				strings.HasPrefix(urlPath, "static/") ||
				strings.HasPrefix(urlPath, "assets/")

			// Check for static file extensions
			if !isStaticFile {
				for _, ext := range staticFileExtensions {
					if strings.HasSuffix(urlPath, ext) {
						isStaticFile = true
						break
					}
				}
			}

			if isStaticFile {
				// This looks like an asset request, return 404
				http.NotFound(w, r)
				return
			}

			// For SPA routes, serve index.html
			if indexFile, err := staticFS.Open("index.html"); err == nil {
				defer safe.Close(r.Context(), indexFile)
				w.Header().Set("Content-Type", "text/html")
				safe.Copy(r.Context(), w, indexFile)
				return
			}

			// If index.html is also not found, return 404
			http.NotFound(w, r)
			return
		} else {
			// File exists, close it and let fileServer handle it
			safe.Close(r.Context(), file)
		}

		// Set appropriate Content-Type for favicon files
		if strings.HasSuffix(urlPath, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
		} else if strings.HasSuffix(urlPath, ".png") && strings.Contains(urlPath, "favicon") {
			w.Header().Set("Content-Type", "image/png")
		}

		// Serve the requested file using the file server
		fileServer.ServeHTTP(w, r)
	}
}
