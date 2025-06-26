package storage_test

import (
	"context"
	"testing"

	"github.com/NailUsmanov/gophermart/internal/mocks"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAddWithdrawOrder(t *testing.T) {

	t.Run("correct operation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockWithdrawLogic(ctrl)
		mockServ.EXPECT().CheckExistOrder(gomock.Any(), "1234567890").Return(false, 0, nil)
		mockServ.EXPECT().GetUserBalance(gomock.Any(), 1).Return(100.0, 0.00, nil)
		mockServ.EXPECT().AddWithdrawOrder(gomock.Any(), 1, "1234567890", 50.0).Return(nil)

		err := ProcessWithdraw(context.Background(), mockServ, 1, "1234567890", 50.0)
		assert.NoError(t, err)
	})

	t.Run("order already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockWithdrawLogic(ctrl)
		mockServ.EXPECT().CheckExistOrder(gomock.Any(), "1234567890").Return(true, 1, nil)

		err := ProcessWithdraw(context.Background(), mockServ, 1, "1234567890", 50.0)

		assert.ErrorIs(t, storage.ErrOrderAlreadyUploaded, err)
	})

	t.Run("not enought money", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockWithdrawLogic(ctrl)
		mockServ.EXPECT().CheckExistOrder(gomock.Any(), "1234567890").Return(false, 0, nil)
		mockServ.EXPECT().GetUserBalance(gomock.Any(), 1).Return(30.0, 10.0, nil)

		err := ProcessWithdraw(context.Background(), mockServ, 1, "1234567890", 50.0)
		assert.ErrorIs(t, err, storage.ErrNotEnoughFunds)
	})
}

func ProcessWithdraw(ctx context.Context, logic storage.WithdrawLogic, userID int, number string, sum float64) error {
	exists, _, err := logic.CheckExistOrder(ctx, number)
	if err != nil {
		return err
	}
	if exists {
		return storage.ErrOrderAlreadyUploaded
	}

	current, withdrawn, err := logic.GetUserBalance(ctx, userID)
	if err != nil {
		return err
	}

	if current-withdrawn < sum {
		return storage.ErrNotEnoughFunds
	}

	return logic.AddWithdrawOrder(ctx, userID, number, sum)
}
