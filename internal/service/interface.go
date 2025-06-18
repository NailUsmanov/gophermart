package service

import (
	"context"

	"github.com/NailUsmanov/gophermart/internal/storage"
	"go.uber.org/zap"
)

type ServiceStorage interface {
	CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error)
	CreateNewOrder(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error
	GetOrdersByUserID(ctx context.Context, userID int) ([]storage.Order, error)
}
