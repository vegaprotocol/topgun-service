package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/leaderboard"
)

type Service struct {
	router *mux.Router
	http   *http.Server
	cfg    config.Config
	lb     *leaderboard.Service
	mu     sync.Mutex
}

func NewAPIService(cfg config.Config, lb *leaderboard.Service) *Service {
	svc := &Service{
		cfg: cfg,
		lb:  lb,
	}
	svc.router = createRoutes(svc)
	svc.http = createServer(svc)
	return svc
}

func createServer(s *Service) *http.Server {
	return &http.Server{
		Addr:         s.cfg.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(s.router),
	}
}

func createRoutes(s *Service) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", s.rootHandler)
	router.HandleFunc("/status", s.statusHandler)
	router.HandleFunc("/leaderboard", s.leaderboardHandler)
	return router
}

func (s *Service) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	content := `<!doctype html>
	<head>
	<title>Topgun Service</title>
	</head>
	<body>
	<h1>Topgun Service</h1>
	<ul>
	<li><a href="/status">Status</a></li>
	<li><a href="/leaderboard">Leaderboard</a></li>
	</ul>
	</body>
	</html>`
	w.Write([]byte(content))
}

func (s *Service) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"success\":true}"))

}

func (s *Service) leaderboardHandler(w http.ResponseWriter, r *http.Request) {

}

func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.http.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
		log.WithError(err).Warn("Failed to serve")
	}
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.http.Close()
}
