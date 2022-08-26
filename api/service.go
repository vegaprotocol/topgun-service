package api

import (
	"context"
	"net/http"
	"strconv"
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

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulShutdownTimeout)
	defer cancel()
	s.http.Shutdown(ctx)
}

func GetQuery(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func GetQueryInt(r *http.Request, key string) int64 {
	q := GetQuery(r, key)
	if len(q) > 0 {
		i, err := strconv.ParseInt(q, 10, 64)
		if err != nil {
			log.Warnf("Could not parse query string param %s %s to int", key, q)
			return -1
		}
		return i
	}
	return -1
}
