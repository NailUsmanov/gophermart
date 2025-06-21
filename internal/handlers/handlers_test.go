package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/models"
	"github.com/NailUsmanov/gophermart/internal/service"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func FakeAuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), middleware.UserLoginKey, 1))
		next.ServeHTTP(w, r)
	})
}

func ptr[T any](v T) *T {
	return &v
}

// Для проверки хендлера Пост Ордер
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

// Для проверки хендлера Гет Ордер
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

// Для проверки хендлеров с балансом
type mockBalanceIndicator struct {
	GetUserBalanceFunc func(ctx context.Context, userID int) (float64, float64, error)
	// Нахождение трат пользователя
	GetUserWithDrawnsFunc func(ctx context.Context, userID int) (float64, error)
	// Добавление суммы списаний в таблицу с заказами
	AddWithdrawOrderFunc func(ctx context.Context, userID int, orderNumber string, sum float64) error
	// Вывод всех списаний конкретного пользователя
	GetAllUserWithdrawalsFunc func(ctx context.Context, userID int) ([]models.UserWithDraw, error)
}

func (mb *mockBalanceIndicator) GetUserBalance(ctx context.Context, userID int) (float64, float64, error) {
	return mb.GetUserBalanceFunc(ctx, userID)
}

func (mb *mockBalanceIndicator) GetUserWithDrawns(ctx context.Context, userID int) (float64, error) {
	return mb.GetUserWithDrawnsFunc(ctx, userID)
}

func (mb *mockBalanceIndicator) AddWithdrawOrder(ctx context.Context, userID int, orderNumber string, sum float64) error {
	return mb.AddWithdrawOrderFunc(ctx, userID, orderNumber, sum)
}

func (mb *mockBalanceIndicator) GetAllUserWithdrawals(ctx context.Context, userID int) ([]models.UserWithDraw, error) {
	return mb.GetAllUserWithdrawalsFunc(ctx, userID)
}

func TestUserBalance(t *testing.T) {

	tests := []struct {
		name        string
		mock        *mockBalanceIndicator
		wantStatus  int
		wantBody    string
		contentType string
	}{
		{
			name: "correct Balance",
			mock: &mockBalanceIndicator{
				GetUserBalanceFunc: func(ctx context.Context, userID int) (float64, float64, error) {
					return 100.00, 0.00, nil
				},
			},
			wantBody:    `{"current":100.00,"withdrawn":0.00}`,
			wantStatus:  http.StatusOK,
			contentType: "application/json",
		},
		{
			name: "internal error with encoding",
			mock: &mockBalanceIndicator{
				GetUserBalanceFunc: func(ctx context.Context, userID int) (float64, float64, error) {
					return 0, 0.00, service.ErrInternal
				},
			},
			wantStatus:  http.StatusInternalServerError,
			wantBody:    "",
			contentType: "text/plain; charset=utf-8",
		},
	}

	logger := zap.NewNop().Sugar()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(FakeAuthMiddleWare)
			r.Get("/api/user/balance", UserBalance(tt.mock, logger))
			// Эмуляция запроса
			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

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

func TestAllUserWithDrawals(t *testing.T) {
	emptyResult := []models.UserWithDraw{}
	correctResult := []models.UserWithDraw{
		{
			NumberOrder: "1234567890",
			Sum:         0.00,
			ProcessedAt: time.Date(2025, 6, 20, 10, 0, 0, 0, time.UTC),
		},
	}
	correctBody := `[{"order":"1234567890","sum":0.00,"processed_at":"2025-06-20T10:00:00Z"}]`
	tests := []struct {
		name        string
		mock        *mockBalanceIndicator
		wantBody    string
		wantStatus  int
		contentType string
		useAuth     bool
	}{
		{
			name: "correct test",
			mock: &mockBalanceIndicator{
				GetAllUserWithdrawalsFunc: func(ctx context.Context, userID int) ([]models.UserWithDraw, error) {
					return correctResult, nil
				},
			},
			wantBody:    correctBody,
			wantStatus:  http.StatusOK,
			contentType: "application/json",
			useAuth:     true,
		},
		{
			name: "empty result",
			mock: &mockBalanceIndicator{
				GetAllUserWithdrawalsFunc: func(ctx context.Context, userID int) ([]models.UserWithDraw, error) {
					return emptyResult, nil
				},
			},
			wantBody:    "",
			wantStatus:  http.StatusNoContent,
			contentType: "",
			useAuth:     true,
		},
		{
			name: "unauthorized user",
			mock: &mockBalanceIndicator{
				GetAllUserWithdrawalsFunc: func(ctx context.Context, userID int) ([]models.UserWithDraw, error) {
					t.Fatal("should not be called for unauthorized request")
					return nil, nil
				},
			},
			wantBody:    "Unauthorized\n",
			wantStatus:  http.StatusUnauthorized,
			contentType: "text/plain; charset=utf-8",
			useAuth:     false,
		},
	}

	logger := zap.NewNop().Sugar()
	for _, tt := range tests {
		r := chi.NewRouter()
		if tt.useAuth {
			r.Use(FakeAuthMiddleWare)
		}
		r.Get("/api/user/withdrawals", AllUserWithDrawals(tt.mock, logger))
		// Эмуляция запроса
		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Проверяем тело только если оно должно быть
		if tt.wantBody != "" {
			if tt.contentType == "application/json" {
				assert.JSONEq(t, tt.wantBody, w.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		}

		// Проверяем Content-Type только если он ожидается
		if tt.contentType != "" {
			assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))
		}
	}
}
