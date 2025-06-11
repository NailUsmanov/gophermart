package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"go.uber.org/zap"
)

func PostOrder(s storage.Storage, sugar *zap.SugaredLogger, v validation.OrderValidation) http.HandlerFunc {
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

		sugar.Infof("raw body for Luhn: %q", orderNum)
		IsValid := v.IsValidLuhn(orderNum)
		sugar.Infof("passed Luhn: %v", IsValid)
		if !IsValid {
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}

		// Достаем номер пользователя
		userID, ok := r.Context().Value(middleware.UserLoginKey).(int)
		sugar.Infof("DEBUG: userID from context = %d (ok=%v)", userID, ok)
		sugar.Infof("DEBUG: context key = %#v", middleware.UserLoginKey)
		sugar.Infof("DEBUG: raw context value = %#v", r.Context().Value(middleware.UserLoginKey))
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем существует ли уже запись в базе
		exists, existingUserID, err := s.CheckExistOrder(r.Context(), orderNum)
		if err != nil {
			sugar.Errorf("ERROR: CheckExistOrder failed: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Если существует, смотрим, принадлежит ли она именно нашему юзеру
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
			sugar.Errorf("CreateNewOrder error: %v", err)
			if errors.Is(err, storage.ErrOrderAlreadyUploaded) {
				http.Error(w, "Order already uploaded by another user", http.StatusConflict)
			} else {
				sugar.Errorf("CheckExistOrder error: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func GetUserOrders(s storage.Storage, sugar *zap.SugaredLogger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sugar.Infof("GetUserOrder endpoint called")

		// Извлекаем UserID из контекста через куки
		userID, ok := r.Context().Value(middleware.UserLoginKey).(int)
		sugar.Infof("DEBUG: userID from context = %d (ok=%v)", userID, ok)
		// Если нет такого юзера возвращаем статус не авторизован
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Получаем все данные по заказам пользователя через метод GetOrdersByUserID
		orders, err := s.GetOrdersByUserID(r.Context(), userID)
		if err != nil {
			sugar.Infof("GetOrdersByUserID failed: %v", err)
			http.Error(w, "Method GetOrders has err", http.StatusInternalServerError)
			return
		}
		// Если нет заказов возвращаем 204 No Content
		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
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
