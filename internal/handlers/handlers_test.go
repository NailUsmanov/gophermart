package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NailUsmanov/gophermart/internal/middleware"
	"github.com/NailUsmanov/gophermart/internal/mocks"
	"github.com/NailUsmanov/gophermart/internal/models"
	"github.com/NailUsmanov/gophermart/internal/service"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/internal/validation"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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

func TestPostOrder(t *testing.T) {

	validator := &validation.LuhnValidation{}
	logger := zap.NewNop().Sugar() // безопасный мок-логгер

	t.Run("correct test", func(t *testing.T) {
		// Создаем контроллер, который следит за исполнением моков.
		// defer ctrl.Finish() проверяет в конце, что все вызовы моков были такими,
		// какими мы ожидали (EXPECT()).
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// mockgen сгенерировал структуру MockServiceInterface — это мок интерфейса ServiceInterface.
		// Мы создаём её экземпляр.
		mockServ := mocks.NewMockServiceInterface(ctrl)
		// Далее вызываем методы, которые используются в хендлере и прописываем, что мы ожиданием от них получить
		mockServ.EXPECT().CheckExistUser(gomock.Any(), "79927398713").Return(false, 0, 1, nil)
		mockServ.EXPECT().CreateNewOrder(gomock.Any(), 1, "79927398713", gomock.Any()).Return(nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Post("/api/user/orders", PostOrder(mockServ, logger, validator))
		// Эмуляция запроса
		req := httptest.NewRequest("POST", "/api/user/orders", strings.NewReader("79927398713"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("order already uploaded by another user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockServiceInterface(ctrl)

		mockServ.EXPECT().
			CheckExistUser(gomock.Any(), "79927398713").
			Return(true, 2, 1, nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Post("/api/user/orders", PostOrder(mockServ, logger, validator))

		req := httptest.NewRequest("POST", "/api/user/orders", strings.NewReader("79927398713"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("invalid order number format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockServiceInterface(ctrl)

		mockServ.EXPECT().
			CheckExistUser(gomock.Any(), "79927398713").
			Return(false, 0, 0, service.ErrInvalidOrderFormat)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Post("/api/user/orders", PostOrder(mockServ, logger, validator))

		req := httptest.NewRequest("POST", "/api/user/orders", strings.NewReader("79927398713"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

// Для проверки хендлера Гет Ордер
func TestGetUserOrders(t *testing.T) {
	// Жестко задаем время (UTC, чтобы было одинаково)
	uploaded := time.Date(2025, 6, 20, 10, 0, 0, 0, time.UTC)
	// Ожидаемый JSON
	userOrder := `[{"number":"1","status":"NEW","uploaded_at":"2025-06-20T10:00:00Z"}]`

	validator := &validation.LuhnValidation{}
	logger := zap.NewNop().Sugar()

	t.Run("correct test", func(t *testing.T) {
		// Создаем контроллер, который следит за исполнением моков.
		// defer ctrl.Finish() проверяет в конце, что все вызовы моков были такими,
		// какими мы ожидали (EXPECT()).
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// mockgen сгенерировал структуру MockServiceInterface — это мок интерфейса ServiceInterface.
		// Мы создаём её экземпляр.
		mockServ := mocks.NewMockServiceStorage(ctrl)

		// Далее вызываем методы, которые используются в хендлере и прописываем, что мы ожиданием от них получить
		// на возврат ожидаем []storage.Order, error.
		mockServ.EXPECT().GetOrdersByUserID(gomock.Any(), 1).Return([]storage.Order{
			{
				Number:     "1",
				Status:     ptr("NEW"),
				Accrual:    nil,
				UploadedAt: uploaded,
			},
		}, nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Get("/api/user/orders", GetUserOrders(mockServ, logger, validator))
		// Эмуляция запроса
		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Проверяем тело
		assert.JSONEq(t, userOrder, w.Body.String())

		// Проверяем Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("no content test", func(t *testing.T) {
		// Создаем контроллер, который следит за исполнением моков.
		// defer ctrl.Finish() проверяет в конце, что все вызовы моков были такими,
		// какими мы ожидали (EXPECT()).
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// mockgen сгенерировал структуру MockServiceInterface — это мок интерфейса ServiceInterface.
		// Мы создаём её экземпляр.
		mockServ := mocks.NewMockServiceStorage(ctrl)

		// Далее вызываем методы, которые используются в хендлере и прописываем, что мы ожиданием от них получить
		// на возврат ожидаем []storage.Order, error.
		mockServ.EXPECT().GetOrdersByUserID(gomock.Any(), 1).Return([]storage.Order{}, nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Get("/api/user/orders", GetUserOrders(mockServ, logger, validator))
		// Эмуляция запроса
		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		// Проверяем Content-Type
		assert.Equal(t, "", w.Header().Get("Content-Type"))
	})
}

// Для проверки хендлеров с балансом
func TestUserBalance(t *testing.T) {
	logger := zap.NewNop().Sugar()
	wantBody := `{"current":100.00,"withdrawn":0.00}`
	t.Run("correct balance", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockBalanceIndicator(ctrl)

		mockServ.EXPECT().GetUserBalance(gomock.Any(), 1).Return(100.00, 0.00, nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Get("/api/user/balance", UserBalance(mockServ, logger))
		// Эмуляция запроса
		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.JSONEq(t, wantBody, w.Body.String())

		// Проверяем Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
	t.Run("internal error with encoding", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockServ := mocks.NewMockBalanceIndicator(ctrl)

		mockServ.EXPECT().GetUserBalance(gomock.Any(), 1).Return(0.00, 0.00, service.ErrInternal)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Get("/api/user/balance", UserBalance(mockServ, logger))
		// Эмуляция запроса
		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Проверяем Content-Type
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})
}

func TestAllUserWithDrawals(t *testing.T) {
	logger := zap.NewNop().Sugar()
	correctResult := []models.UserWithDraw{
		{
			NumberOrder: "1234567890",
			Sum:         0.00,
			ProcessedAt: time.Date(2025, 6, 20, 10, 0, 0, 0, time.UTC),
		},
	}
	correctBody := `[{"order":"1234567890","sum":0.00,"processed_at":"2025-06-20T10:00:00Z"}]`

	t.Run("correct test", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := mocks.NewMockBalanceIndicator(ctrl)
		mock.EXPECT().GetAllUserWithdrawals(gomock.Any(), 1).Return(correctResult, nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Get("/api/user/withdrawals", AllUserWithDrawals(mock, logger))

		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, correctBody, w.Body.String())
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := mocks.NewMockBalanceIndicator(ctrl)
		mock.EXPECT().GetAllUserWithdrawals(gomock.Any(), 1).Return([]models.UserWithDraw{}, nil)

		r := chi.NewRouter()
		r.Use(FakeAuthMiddleWare)
		r.Get("/api/user/withdrawals", AllUserWithDrawals(mock, logger))

		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "", w.Header().Get("Content-Type"))
	})

	t.Run("unauthorized user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := mocks.NewMockBalanceIndicator(ctrl)

		r := chi.NewRouter() // Без FakeAuthMiddleWare — неавторизован
		r.Get("/api/user/withdrawals", AllUserWithDrawals(mock, logger))

		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "Unauthorized\n", w.Body.String())
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})
}
