package storage

import (
	"context"
	"errors"

	"github.com/NailUsmanov/gophermart/internal/interfaces"
	"github.com/NailUsmanov/gophermart/internal/models"
	"go.uber.org/zap"
)

var ErrOrderAlreadyUsed = errors.New("order number already used")
var ErrOrderAlreadyUploaded = errors.New("order already uploaded by another person")
var ErrNotEnoughFunds = errors.New("insufficient funds")

type OrderOption interface {
	CreateNewOrder(ctx context.Context, userNumber int, numberOrder string, sugar *zap.SugaredLogger) error
	CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]Order, error)
}

type WorkerAccrual interface {
	// GetOrdersForAccrualUpdate возвращает все заказы со статусами NEW, PROCESSING,
	// REGISTERED для обновления статуса и начислений
	GetOrdersForAccrualUpdate(ctx context.Context) ([]Order, error)

	// UpdateOrderStatus обновляет статус и сумму начислений по номеру заказа
	// (используется воркером после запроса к accrual-системе)
	UpdateOrderStatus(ctx context.Context, number string, status string, accrual *float64) error
}
type BalanceIndicator interface {
	// Для показаний текущего баланса и трат предыдущих
	GetUserBalance(ctx context.Context, userID int) (float64, float64, error)
	// Нахождение трат пользователя
	GetUserWithDrawns(ctx context.Context, userID int) (float64, error)
	// Добавление суммы списаний в таблицу с заказами
	AddWithdrawOrder(ctx context.Context, userID int, orderNumber string, sum float64) error
	// Вывод всех списаний конкретного пользователя
	GetAllUserWithdrawals(ctx context.Context, userID int) ([]models.UserWithDraw, error)
}

// Создаем интерфейс для работы с ним в тестах
type WithdrawLogic interface {
	CheckExistOrder(ctx context.Context, numberOrder string) (bool, int, error)
	GetUserBalance(ctx context.Context, userID int) (float64, float64, error)
	AddWithdrawOrder(ctx context.Context, userID int, number string, sum float64) error
}

// Только для хендлера AllUserWithdrawals
type WithdrawalFetcher interface {
	GetAllUserWithdrawals(ctx context.Context, userID int) ([]models.UserWithDraw, error)
}

type Storage interface {
	WithdrawLogic
	interfaces.Auth
	OrderOption
	WorkerAccrual
	BalanceIndicator
	WithdrawalFetcher
}
