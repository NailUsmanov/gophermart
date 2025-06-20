package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type mockService struct {
	CheckExistUserFunc func(ctx context.Context, orderNum string) (bool, int, int, error)
	CreateNewOrderFunc func(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error
}

func (m *mockService) CheckExistUser(ctx context.Context, orderNum string) (bool, int, int, error) {
	return m.CheckExistUserFunc(ctx, orderNum)
}

func (m *mockService) CreateNewOrder(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
	return m.CreateNewOrderFunc(ctx, userID, orderNum, sugar)
}

func FakeAuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), middleware.UserLoginKey, 1))
		next.ServeHTTP(w, r)
	})
}

// успешная отправка
func TestPostOrder(t *testing.T) {
	mock := &mockService{
		CheckExistUserFunc: func(ctx context.Context, numberOrder string) (bool, int, int, error) {
			return false, 0, 1, nil // exists, existingUserID, userID, err
		},
		CreateNewOrderFunc: func(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
			return nil
		},
	}
	validator := &validation.LuhnValidation{}
	logger := zap.NewNop().Sugar() // безопасный мок-логгер
	r := chi.NewRouter()
	r.Use(FakeAuthMiddleWare)
	r.Post("/api/user/orders", PostOrder(mock, logger, validator))

	// Эмуляция запроса
	req := httptest.NewRequest("POST", "/api/user/orders", strings.NewReader("79927398713"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}

}
