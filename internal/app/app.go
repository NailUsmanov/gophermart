package app

import (
	"context"
	"net/http"

	"github.com/NailUsmanov/gophermart/internal/handlers"
	"github.com/NailUsmanov/gophermart/internal/interfaces"
	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"github.com/NailUsmanov/gophermart/internal/worker"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type App struct {
	storage    storage.Storage
	router     *chi.Mux
	sugar      *zap.SugaredLogger
	worker     *worker.Worker
	validation *validation.LuhnValidation
}

func NewApp(s storage.Storage, sugar *zap.SugaredLogger, accrualHost string) *App {
	r := chi.NewRouter()
	w := worker.NewWorker(s, sugar, accrualHost)
	v := validation.LuhnValidation{}
	app := &App{
		storage:    s,
		router:     r,
		sugar:      sugar,
		worker:     w,
		validation: &v,
	}
	sugar.Info("App initialized")
	w.Start(context.Background())
	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {
	a.router.Use(middleware.LoggingMiddleWare(a.sugar))
	auth := interfaces.Auth(a.storage)
	a.router.Post("/api/user/register", handlers.Register(auth, a.sugar))
	a.router.Post("/api/user/login", handlers.Login(auth, a.sugar))

	a.router.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Use(middleware.GzipMiddleware)
		r.Post("/orders", handlers.PostOrder(a.storage, a.sugar, a.validation))
		r.Get("/orders", handlers.GetUserOrders(a.storage, a.sugar, a.validation))
		r.Get("/balance", handlers.UserBalance(a.storage, a.sugar))
		r.Post("/balance/withdraw", handlers.WithDraw(a.storage, a.sugar, a.validation))
		r.Get("/withdrawals", handlers.AllUserWithDrawals(a.storage, a.sugar))
	})
}
func (a *App) Run(ctx context.Context, addr string) error {
	srv := http.Server{
		Addr:    addr,
		Handler: a.router,
	}
	// В фоновом потоке ждём, пока контекст не будет отменён (через cancel() в main)
	go func() {
		<-ctx.Done()
		a.sugar.Infow("Shutting down server...")
		// Graceful shutdown: останавливаем HTTP-сервер, завершаем текущие соединения, новые не принимаем.
		_ = srv.Shutdown(context.Background())
	}()
	return srv.ListenAndServe()
}
