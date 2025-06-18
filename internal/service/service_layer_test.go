package service

import (
	"context"
	"errors"
	"testing"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/NailUsmanov/gophermart/internal/validation"
	_ "golang.org/x/crypto/nacl/auth"
)

type mockStorage struct {
	exists         bool
	existingUserID int
	err            error
}

func (m *mockStorage) CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error) {
	return m.exists, m.existingUserID, m.err
}

// Заглушки для других методов (если компилятор будет требовать реализации полного интерфейса):
func (m *mockStorage) CreateNewOrder(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
	return nil
}
func (m *mockStorage) GetUserOrders(ctx context.Context) ([]storage.Order, error) {
	return nil, nil
}
func (m *mockStorage) GetOrdersByUserID(ctx context.Context, userID int) ([]storage.Order, error) {
	return nil, nil
}
func (m *mockStorage) AddWithdrawOrder(ctx context.Context, userID int, orderNumber string, sum float64) error {
	return nil
}
func TestCheckExistUser(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.UserLoginKey, 1)

	tests := []struct {
		name           string
		orderNum       string
		storageMock    *mockStorage
		expectedExists bool
		expectedErr    error
	}{
		{
			name:     "valid order, user exists",
			orderNum: "79927398713",
			storageMock: &mockStorage{
				exists: true, existingUserID: 1, err: nil,
			},
			expectedExists: true,
			expectedErr:    nil,
		},
		{
			name:     "invalid order format",
			orderNum: "invalid-order",
			storageMock: &mockStorage{
				exists: false, existingUserID: 0, err: nil,
			},
			expectedExists: false,
			expectedErr:    ErrInvalidOrderFormat,
		},
		{name: "storage error",
			orderNum: "79927398713",
			storageMock: &mockStorage{
				exists: false, existingUserID: 0, err: errors.New("some error"),
			},
			expectedExists: false,
			expectedErr:    ErrInternal,
		},
	}
	validation := &validation.LuhnValidation{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.storageMock, validation)
			exists, _, _, err := service.CheckExistUser(ctx, tt.orderNum)
			assert.Equal(t, tt.expectedExists, exists)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
