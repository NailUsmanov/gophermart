package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/NailUsmanov/gophermart/internal/storage"
	"go.uber.org/zap"
)

type Worker struct {
	Storage     storage.Storage
	Sugar       *zap.SugaredLogger
	AccrualHost string
}

func NewWorker(storage storage.Storage, sugar *zap.SugaredLogger, acrrualHost string) *Worker {
	return &Worker{
		Storage:     storage,
		Sugar:       sugar,
		AccrualHost: acrrualHost,
	}
}

// Создаем метод StartWorker для фонового вызова воркера с проверкой статуса заказа из Аккруал хендлера
func (w *Worker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				w.Sugar.Info("Worker stopped due to context cancellation")
				return
			case <-ticker.C:
				orders, err := w.Storage.GetOrdersForAccrualUpdate(ctx)
				w.Sugar.Infof(">>> Worker tick: found %d orders", len(orders))
				if err != nil {
					w.Sugar.Errorf("Method GetOrdersForAccrualUpdate has err: %v", err)
					continue
				}
				for _, order := range orders {
					// GET запрос в accrual систему: http://{accrualHost}/api/orders/{order.Number}
					url := fmt.Sprintf("%s/api/orders/%s", w.AccrualHost, order.Number)
					resp, err := http.Get(url)
					if err != nil {
						w.Sugar.Errorf("HTTP GET failed for %s: %v", url, err)
						continue
					}
					func() {
						defer resp.Body.Close()
						// Обработка ответа
						if resp.StatusCode == http.StatusNoContent {
							return
						}
						// В случае, когда превышено количество запросов, ждем время,
						// которое указно в хедере Retry - After
						if resp.StatusCode == http.StatusTooManyRequests {
							retryAfter := resp.Header.Get("Retry-After")
							if sec, err := strconv.Atoi(retryAfter); err == nil && sec > 0 {
								w.Sugar.Warnf("Too many requests, sleeping for %d seconds", sec)
								time.Sleep(time.Duration(sec) * time.Second)
							}
							return
						}
						if resp.StatusCode != http.StatusOK {
							w.Sugar.Warnf("Unexpected status from accrual: %d", resp.StatusCode)
							return
						}
						// Создаем структуру аккруал, в которую дальше будем декодировать данные из тела ответа JSON
						var accrualResp struct {
							Order   string   `json:"order"`
							Status  string   `json:"status"`
							Accrual *float64 `json:"accrual,omitempty"`
						}
						if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
							w.Sugar.Errorf("Failed to decode accrual response: %v", err)
							return
						}
						// Вызываем метод для обновления данных
						err = w.Storage.UpdateOrderStatus(ctx, accrualResp.Order, accrualResp.Status, accrualResp.Accrual)
						if err != nil {
							w.Sugar.Errorf("UpdateOrderStatus failed: %v", err)
							return
						}
						w.Sugar.Infof("Updated order %s to %s", accrualResp.Order, accrualResp.Status)
					}()
				}
			}
		}
	}()

}
