package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// QueueBroker управляет очередями и сообщениями
type QueueBroker struct {
	queues         map[string]chan string
	maxQueueSize   int
	maxQueues      int
	defaultTimeout int
	mu             sync.Mutex
}

// NewQueueBroker создает новый экземпляр QueueBroker
func NewQueueBroker(maxQueueSize, maxQueues, defaultTimeout int) *QueueBroker {
	return &QueueBroker{
		queues:         make(map[string]chan string),
		maxQueueSize:   maxQueueSize,
		maxQueues:      maxQueues,
		defaultTimeout: defaultTimeout,
	}
}

// PutMessage добавляет сообщение в очередь
func (qb *QueueBroker) PutMessage(queueName, message string) error {
	qb.mu.Lock()
	defer qb.mu.Unlock()

	if len(qb.queues) >= qb.maxQueues && qb.queues[queueName] == nil {
		return errors.New("maximum number of queues reached")
	}

	if qb.queues[queueName] == nil {
		qb.queues[queueName] = make(chan string, qb.maxQueueSize)
	}

	select {
	case qb.queues[queueName] <- message:
		return nil
	default:
		return errors.New("queue is full")
	}
}

// GetMessage извлекает сообщение из очереди
func (qb *QueueBroker) GetMessage(queueName string, timeout int) (string, error) {
	qb.mu.Lock()
	queue, exists := qb.queues[queueName]
	qb.mu.Unlock()

	if !exists {
		return "", errors.New("queue does not exist")
	}

	select {
	case message := <-queue:
		return message, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		return "", errors.New("not found")
	}
}

// QueueHandler обрабатывает HTTP-запросы
func QueueHandler(qb *QueueBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlePut(qb, w, r)
		case http.MethodGet:
			handleGet(qb, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handlePut обрабатывает PUT-запросы
func handlePut(qb *QueueBroker, w http.ResponseWriter, r *http.Request) {
	queueName := r.URL.Path[len("/queue/"):]
	if queueName == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var requestBody struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil || requestBody.Message == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := qb.PutMessage(queueName, requestBody.Message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleGet обрабатывает GET-запросы
func handleGet(qb *QueueBroker, w http.ResponseWriter, r *http.Request) {
	queueName := r.URL.Path[len("/queue/"):]
	if queueName == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	timeout := qb.defaultTimeout
	if timeoutParam := r.URL.Query().Get("timeout"); timeoutParam != "" {
		var err error
		timeout, err = strconv.Atoi(timeoutParam)
		if err != nil || timeout < 0 {
			http.Error(w, "Invalid timeout", http.StatusBadRequest)
			return
		}
	}

	message, err := qb.GetMessage(queueName, timeout)
	if err != nil {
		if err.Error() == "not found" {
			http.Error(w, "Not found", http.StatusNotFound)
		} else if err.Error() == "queue does not exist" {
			http.Error(w, "Queue does not exist", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func main() {
	// Парсинг аргументов командной строки
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Usage: ./queue_broker --port <port> --max-queue-size <size> --max-queues <count> --default-timeout <timeout>")
		return
	}

	port := 8080
	maxQueueSize := 100
	maxQueues := 10
	defaultTimeout := 10

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			port, _ = strconv.Atoi(args[i+1])
		case "--max-queue-size":
			maxQueueSize, _ = strconv.Atoi(args[i+1])
		case "--max-queues":
			maxQueues, _ = strconv.Atoi(args[i+1])
		case "--default-timeout":
			defaultTimeout, _ = strconv.Atoi(args[i+1])
		}
	}

	// Создание и запуск сервера
	qb := NewQueueBroker(maxQueueSize, maxQueues, defaultTimeout)
	http.Handle("/queue/", QueueHandler(qb))

	fmt.Printf("Starting server on port %d...\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
