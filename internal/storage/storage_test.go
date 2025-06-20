package storage

import (
	"context"
	"testing"
)

type mockWithdrawLogic struct {
	CheckExistOrderFunc  func(context.Context, string) (bool, int, error)
	GetUserBalanceFunc   func(context.Context, int) (float64, float64, error)
	AddWithdrawOrderFunc func(context.Context, int, string, float64) error
}

func (m *mockWithdrawLogic) CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error) {
	return m.CheckExistOrderFunc(ctx, numberOrder)
}

func (m *mockWithdrawLogic) GetUserBalance(ctx context.Context, userID int) (float64, float64, error) {
	return m.GetUserBalanceFunc(ctx, userID)
}

func (m *mockWithdrawLogic) AddWithdrawOrder(ctx context.Context, userID int, number string, sum float64) error {
	return m.AddWithdrawOrderFunc(ctx, userID, number, sum)
}

func TestAddWithdrawOrder(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockWithdrawLogic
		wantErr error
	}{
		{
			name: "not enought money",
			mock: &mockWithdrawLogic{
				CheckExistOrderFunc: func(_ context.Context, _ string) (bool, int, error) {
					return false, 0, nil
				},
				GetUserBalanceFunc: func(_ context.Context, _ int) (float64, float64, error) {
					return 100.0, 0, nil
				},
				AddWithdrawOrderFunc: func(_ context.Context, _ int, _ string, _ float64) error {
					return ErrNotEnoughFunds
				},
			},
			wantErr: ErrNotEnoughFunds,
		},
		{
			name: "order already exists",
			mock: &mockWithdrawLogic{
				CheckExistOrderFunc: func(_ context.Context, _ string) (bool, int, error) {
					return true, 1, ErrOrderAlreadyUploaded
				},
				GetUserBalanceFunc: func(_ context.Context, _ int) (float64, float64, error) {
					return 100.0, 0, nil
				},
				AddWithdrawOrderFunc: func(_ context.Context, _ int, _ string, _ float64) error {
					return ErrOrderAlreadyUploaded
				},
			},
			wantErr: ErrOrderAlreadyUploaded,
		},
		{
			name: "correct operation",
			mock: &mockWithdrawLogic{
				CheckExistOrderFunc: func(_ context.Context, _ string) (bool, int, error) {
					return false, 0, nil
				},
				GetUserBalanceFunc: func(_ context.Context, _ int) (float64, float64, error) {
					return 100.0, 0, nil
				},
				AddWithdrawOrderFunc: func(_ context.Context, _ int, _ string, _ float64) error {
					return nil
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mock.AddWithdrawOrder(context.Background(), 1, "1234567890", 50)
			if err != tt.wantErr {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})

	}
}
