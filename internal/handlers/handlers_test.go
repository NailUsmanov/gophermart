package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/service"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockService struct {
	CheckExistUserFunc func(ctx context.Context, orderNum string) (bool, int, int, error)
	CreateNewOrderFunc func(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error
}

type mockServiceGetOrder struct {
	GetOrdersByUserIDFunc func(ctx context.Context, userID int) ([]storage.Order, error)
}

func (mg *mockServiceGetOrder) GetOrdersByUserID(ctx context.Context, userID int) ([]storage.Order, error) {
	return mg.GetOrdersByUserIDFunc(ctx, userID)
}
func (mg *mockServiceGetOrder) CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error) {
	return false, 0, nil // заглушка
}

func (mg *mockServiceGetOrder) CreateNewOrder(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
	return nil // заглушка
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

func ptr[T any](v T) *T {
	return &v
}

// успешная отправка
func TestPostOrder(t *testing.T) {
	tests := []struct {
		name       string
		mock       *mockService
		wantStatus int
	}{
		{
			name: "correct test",
			mock: &mockService{
				CheckExistUserFunc: func(ctx context.Context, numberOrder string) (bool, int, int, error) {
					return false, 0, 1, nil // exists, existingUserID, userID, err
				},
				CreateNewOrderFunc: func(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
					return nil
				},
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name: "Order already uploaded by another user",
			mock: &mockService{
				CheckExistUserFunc: func(ctx context.Context, numberOrder string) (bool, int, int, error) {
					return true, 2, 1, nil // exists, existingUserID, userID, err
				},
				CreateNewOrderFunc: func(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
					return nil
				},
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "Invalid order number format",
			mock: &mockService{
				CheckExistUserFunc: func(ctx context.Context, numberOrder string) (bool, int, int, error) {
					return false, 0, 0, service.ErrInvalidOrderFormat // exists, existingUserID, userID, err
				},
				CreateNewOrderFunc: func(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
					return service.ErrInvalidOrderFormat
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	validator := &validation.LuhnValidation{}
	logger := zap.NewNop().Sugar() // безопасный мок-логгер

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(FakeAuthMiddleWare)
			r.Post("/api/user/orders", PostOrder(tt.mock, logger, validator))
			// Эмуляция запроса
			req := httptest.NewRequest("POST", "/api/user/orders", strings.NewReader("79927398713"))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}

}

func TestGetUserOrders(t *testing.T) {
	// Жестко задаем время (UTC, чтобы было одинаково)
	uploaded := time.Date(2025, 6, 20, 10, 0, 0, 0, time.UTC)
	// Ожидаемый JSON
	userOrder := `[{"number":"1","status":"NEW","uploaded_at":"2025-06-20T10:00:00Z"}]`

	tests := []struct {
		name        string
		mock        *mockServiceGetOrder
		wantStatus  int
		wantBody    string
		contentType string
	}{
		{
			name: "correct test",
			mock: &mockServiceGetOrder{
				GetOrdersByUserIDFunc: func(ctx context.Context, userID int) ([]storage.Order, error) {
					return []storage.Order{
						{
							Number:     "1",
							Status:     ptr("NEW"),
							Accrual:    nil,
							UploadedAt: uploaded,
						},
					}, nil
				},
			},
			wantStatus:  http.StatusOK,
			wantBody:    userOrder,
			contentType: "application/json",
		},
		{
			name: "no content",
			mock: &mockServiceGetOrder{
				GetOrdersByUserIDFunc: func(ctx context.Context, userID int) ([]storage.Order, error) {
					return []storage.Order{}, nil
				},
			},
			wantStatus:  http.StatusNoContent,
			contentType: "",
		},
	}

	validator := &validation.LuhnValidation{}
	logger := zap.NewNop().Sugar()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(FakeAuthMiddleWare)
			r.Get("/api/user/orders", GetUserOrders(tt.mock, logger, validator))
			// Эмуляция запроса
			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			// Проверяем тело только если оно должно быть
			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, w.Body.String())
			}

			// Проверяем Content-Type только если он ожидается
			if tt.contentType != "" {
				assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))
			}
		})
	}
}
