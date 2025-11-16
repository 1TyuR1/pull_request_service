package http

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	pgrepo "avito/internal/repository/postgres"
	"avito/internal/service"
)

func NewRouter(db *sql.DB) http.Handler {
	r := chi.NewRouter()

	// middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// репозитории Postgres
	teamRepo := pgrepo.NewTeamRepo(db)
	userRepo := pgrepo.NewUserRepo(db)
	prRepo := pgrepo.NewPRRepo(db)

	// сервисы
	teamSvc := service.NewTeamService(teamRepo, userRepo, prRepo)
	userSvc := service.NewUserService(userRepo)
	prSvc := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// инжектим prSvc обратно в teamSvc для BulkDeactivateTeam
	teamSvc.SetPullRequestService(prSvc)

	// хендлеры
	teamHandler := NewTeamHandler(teamSvc)
	userHandler := NewUserHandler(userSvc)
	prHandler := NewPullRequestHandler(prSvc)
	statsHandler := NewStatsHandler(prSvc)

	// health-check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// /team/*
	r.Route("/team", func(r chi.Router) {
		r.Post("/add", teamHandler.AddTeam)
		r.Get("/get", teamHandler.GetTeam)
		r.Post("/deactivateUsers", teamHandler.BulkDeactivate)
	})

	// /users/*
	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", userHandler.SetIsActive)
		r.Get("/getReview", prHandler.GetUserReviews)
	})

	// /pullRequest/*
	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", prHandler.Create)
		r.Post("/merge", prHandler.Merge)
		r.Post("/reassign", prHandler.Reassign)
	})

	// эндпоинт статистики
	r.Get("/stats", statsHandler.GetStats)

	return r
}

func NewRouterForTest(
	teamSvc *service.TeamService,
	userSvc *service.UserService,
	prSvc *service.PullRequestService,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	teamHandler := NewTeamHandler(teamSvc)
	userHandler := NewUserHandler(userSvc)
	prHandler := NewPullRequestHandler(prSvc)
	statsHandler := NewStatsHandler(prSvc)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", teamHandler.AddTeam)
		r.Get("/get", teamHandler.GetTeam)
		r.Post("/deactivateUsers", teamHandler.BulkDeactivate)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", userHandler.SetIsActive)
		r.Get("/getReview", prHandler.GetUserReviews)
	})

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", prHandler.Create)
		r.Post("/merge", prHandler.Merge)
		r.Post("/reassign", prHandler.Reassign)
	})

	r.Get("/stats", statsHandler.GetStats)

	return r
}
