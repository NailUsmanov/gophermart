package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NailUsmanov/gophermart/internal/models"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"go.uber.org/zap"
)

func NewTestLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{}
	logger, _ := cfg.Build()
	return logger.Sugar()
}

type mockStorage struct{}

func (m *mockStorage) Registration(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockStorage) GetUserByLogin(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockStorage) GetUserIDByLogin(_ context.Context, _ string) (int, error) {
	return 1, nil
}

func (m *mockStorage) CheckHashMatch(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockStorage) AddWithdrawOrder(ctx context.Context, userID int, orderNumber string, sum float64) error {
	return nil
}
func (m *mockStorage) GetAllUserWithdrawals(ctx context.Context, userID int) ([]models.UserWithDraw, error) {
	return nil, nil
}

func (m *mockStorage) GetOrdersByUserID(ctx context.Context, userID int) ([]storage.Order, error) {
	return nil, nil
}

func (m *mockStorage) CreateNewOrder(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
	return nil
}

func (m *mockStorage) CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error) {
	return false, 0, nil
}

func (m *mockStorage) GetOrdersForAccrualUpdate(ctx context.Context) ([]storage.Order, error) {
	return nil, nil
}

func (m *mockStorage) UpdateOrderStatus(ctx context.Context, number string, status string, accrual *float64) error {
	return nil
}

func (m *mockStorage) GetUserBalance(ctx context.Context, userID int) (float64, float64, error) {
	return 0, 0, nil
}

func (m *mockStorage) GetUserWithDrawns(ctx context.Context, userID int) (float64, error) {
	return 0, nil
}

func TestNewApp_InitializesRoutes(t *testing.T) {
	sugar := NewTestLogger()
	app := NewApp(&mockStorage{}, sugar, "http://localhost:8080")

	req := httptest.NewRequest("POST", "/api/user/register", nil)
	w := httptest.NewRecorder()

	app.router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Errorf("Маршрут /api/user/register не найден")
	}
}
