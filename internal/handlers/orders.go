package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/NailUsmanov/gophermart/internal/service"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"go.uber.org/zap"
)

func PostOrder(s service.ServiceInterface, sugar *zap.SugaredLogger, v validation.OrderValidation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sugar.Infof(">>> PostOrder endpoint called")
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Invalid content-type", http.StatusBadRequest)
			return
		}
		sugar.Infof("Content-Type: %s", r.Header.Get("Content-Type"))

		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		sugar.Infof("Received request body: %q", body)
		orderNum := string(body)

		// Проверяем валидность заказа через алгоритм Луна или выдаем ошибку 422 -  неверный формат номера заказа;
		// Достаем номер пользователя
		sugar.Infof("raw body for Luhn: %q", orderNum)

		exists, existingUserID, userID, err := s.CheckExistUser(r.Context(), orderNum)
		if err != nil {
			switch err {
			case service.ErrUnauthorized:
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			case service.ErrInvalidOrderFormat:
				http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			case service.ErrInternal:
				http.Error(w, "internal server error", http.StatusInternalServerError)
			default:
				http.Error(w, "unknown error", http.StatusInternalServerError)
			}
			return
		}
		// Проверяем существует ли уже запись в базе
		if exists {
			if existingUserID == userID {
				w.WriteHeader(http.StatusOK)
			} else {
				http.Error(w, "Order already uploaded by another user", http.StatusConflict)
			}
			return
		}

		// Создаем новый заказ. Если заказ уже существует по такому номеру, то вернет ошибку
		sugar.Infof("Calling CreateNewOrder with userID=%d, orderNum=%s", userID, orderNum)
		if err := s.CreateNewOrder(r.Context(), userID, orderNum, sugar); err != nil {
			switch err {
			case service.ErrInternal:
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			case storage.ErrOrderAlreadyUploaded:
				http.Error(w, "order already uploaded by another person", http.StatusConflict)
			}
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func GetUserOrders(s storage.Storage, sugar *zap.SugaredLogger, v validation.OrderValidation) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sugar.Infof("GetUserOrder endpoint called")

		// Создаем структуру сервис слоя
		serv := service.NewService(s, v)

		// Получаем все данные по заказам пользователя через метод GetOrdersByUserID
		// и проверяем на наличие записей по конкретному пользователю
		orders, err := serv.GetUserOrders(r.Context())
		if err != nil {
			sugar.Infof("GetOrdersByUserID failed: %v", err)
			switch err {
			case service.ErrUnauthorized:
				http.Error(w, "Unauthorize", http.StatusUnauthorized)
			case service.ErrNoContent:
				w.WriteHeader(http.StatusNoContent)
			case service.ErrInternal:
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}
		// Если все ок, то возвращаем JSON со списком заказов
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		if err := enc.Encode(orders); err != nil {
			sugar.Error("error encoding response")
			http.Error(w, "error with encoding response", http.StatusInternalServerError)
			return
		}
	})
}
