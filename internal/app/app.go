package app

import (
	"context"
	"net/http"

	"github.com/NailUsmanov/gophermart/internal/handlers"
	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/worker"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type App struct {
	storage storage.Storage
	router  *chi.Mux
	sugar   *zap.SugaredLogger
	worker  *worker.Worker
}

func NewApp(s storage.Storage, sugar *zap.SugaredLogger, accrualHost string) *App {
	r := chi.NewRouter()
	w := worker.NewWorker(s, sugar, accrualHost)
	app := &App{
		storage: s,
		router:  r,
		sugar:   sugar,
		worker:  w,
	}
	sugar.Info("App initialized")
	w.Start(context.Background())
	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {
	a.router.Use(middleware.LoggingMiddleWare(a.sugar))
	a.router.Post("/api/user/register", handlers.Register(a.storage, a.sugar))
	a.router.Post("/api/user/login", handlers.Login(a.storage, a.sugar))

	a.router.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Use(middleware.GzipMiddleware)
		r.Post("/orders", handlers.PostOrder(a.storage, a.sugar))
		r.Get("/orders", handlers.GetUserOrders(a.storage, a.sugar))
		r.Get("/balance", handlers.UserBalance(a.storage, a.sugar))
		r.Post("/balance/withdraw", handlers.WithDraw(a.storage, a.sugar))
		r.Get("/withdrawals", handlers.AllUserWithDrawals(a.storage, a.sugar))
	})
}
func (a *App) Run(addr string) error {
	return http.ListenAndServe(addr, a.router)
}
