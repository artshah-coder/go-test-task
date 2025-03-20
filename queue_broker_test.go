package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPutMessage проверяет корректность добавления сообщения в очередь
func TestPutMessage(t *testing.T) {
	qb := NewQueueBroker(100, 10, 10)

	// Создаем тестовый HTTP-запрос
	body := map[string]string{"message": "test message"}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("PUT", "/queue/testQueue", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()
	handler := QueueHandler(qb)

	// Выполняем запрос
	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Проверяем, что сообщение добавлено в очередь
	message, err := qb.GetMessage("testQueue", 1)
	if err != nil || message != "test message" {
		t.Errorf("message was not added to the queue: %v", err)
	}
}

// TestPutMessageInvalidBody проверяет обработку некорректного тела запроса
func TestPutMessageInvalidBody(t *testing.T) {
	qb := NewQueueBroker(100, 10, 10)

	// Создаем тестовый HTTP-запрос с некорректным телом
	req, err := http.NewRequest("PUT", "/queue/testQueue", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := QueueHandler(qb)

	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestGetMessage проверяет корректность извлечения сообщения из очереди
func TestGetMessage(t *testing.T) {
	qb := NewQueueBroker(100, 10, 10)

	// Добавляем сообщение в очередь
	qb.PutMessage("testQueue", "test message")

	// Создаем тестовый HTTP-запрос
	req, err := http.NewRequest("GET", "/queue/testQueue", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := QueueHandler(qb)

	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Проверяем тело ответа
	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["message"] != "test message" {
		t.Errorf("handler returned unexpected body: got %v want %v", response["message"], "test message")
	}
}

// TestGetMessageTimeout проверяет обработку таймаута при извлечении сообщения
func TestGetMessageTimeout(t *testing.T) {
	qb := NewQueueBroker(100, 10, 1) // Таймаут 1 секунда

	// Создаем тестовый HTTP-запрос с таймаутом
	req, err := http.NewRequest("GET", "/queue/testQueue?timeout=1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := QueueHandler(qb)

	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

// TestGetMessageNonexistentQueue проверяет обработку запроса к несуществующей очереди
func TestGetMessageNonexistentQueue(t *testing.T) {
	qb := NewQueueBroker(100, 10, 10)

	// Создаем тестовый HTTP-запрос к несуществующей очереди
	req, err := http.NewRequest("GET", "/queue/nonexistentQueue", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := QueueHandler(qb)

	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestPutMessageMaxQueues проверяет обработку превышения максимального количества очередей
func TestPutMessageMaxQueues(t *testing.T) {
	qb := NewQueueBroker(100, 1, 10) // Максимум 1 очередь

	// Добавляем первую очередь
	qb.PutMessage("queue1", "message1")

	// Пытаемся добавить вторую очередь
	body := map[string]string{"message": "message2"}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("PUT", "/queue/queue2", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := QueueHandler(qb)

	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
