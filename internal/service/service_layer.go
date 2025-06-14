package service

import (
	"context"
	"errors"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"go.uber.org/zap"
)

var (
	ErrInvalidOrderFormat = errors.New("invalid order number format")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInternal           = errors.New("internal server error")
	ErrNoContent          = errors.New("no content inside")
)

type Service struct {
	Storage   storage.Storage
	Validator validation.OrderValidation
}

func NewService(s storage.Storage, v validation.OrderValidation) *Service {
	return &Service{
		Storage:   s,
		Validator: v,
	}
}

func (s *Service) CheckExistUser(ctx context.Context, orderNum string) (exists bool, existingUser int, userID int, err error) {
	// Проверяем валидность заказа через алгоритм Луна или выдаем ошибку 422 -  неверный формат номера заказа;
	IsValid := s.Validator.IsValidLuhn(orderNum)
	if !IsValid {
		return false, 0, 0, ErrInvalidOrderFormat
	}
	// Достаем номер пользователя
	userID, ok := ctx.Value(middleware.UserLoginKey).(int)
	if !ok {
		return false, 0, 0, ErrUnauthorized
	}
	// Проверяем существует ли уже запись в базе
	exists, existingUserID, err := s.Storage.CheckExistOrder(ctx, orderNum)
	if err != nil {
		return false, 0, 0, ErrInternal
	}
	return exists, existingUserID, userID, nil
}

func (s *Service) CreateNewOrder(ctx context.Context, userID int, orderNum string, sugar *zap.SugaredLogger) error {
	if err := s.Storage.CreateNewOrder(ctx, userID, orderNum, sugar); err != nil {
		if errors.Is(err, storage.ErrOrderAlreadyUploaded) {
			return storage.ErrOrderAlreadyUploaded // statusConflict
		} else {
			return ErrInternal
		}
	}
	return nil
}

func (s *Service) GetUserOrders(ctx context.Context) ([]storage.Order, error) {
	// Извлекаем UserID из контекста через куки
	userID, ok := ctx.Value(middleware.UserLoginKey).(int)
	// Если нет такого юзера возвращаем статус не авторизован
	if !ok {
		return nil, ErrUnauthorized
	}
	// Используем метод для получения всех заказов
	orders, err := s.Storage.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, ErrInternal
	}
	if len(orders) == 0 {
		return nil, ErrNoContent
	}
	return orders, nil
}
