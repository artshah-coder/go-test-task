# Тестовое задание 

## Очередь на go с REST интерфейсом

# Запуск:
```
go run queue_broker.go --port 8080 --max-queue-size 100 --max-queues 10 --default-timeout 10
```

# Примеры запросов:

1. PUT:
```
curl -X PUT -H "Content-Type: application/json" -d '{"message": "data"}' http://localhost:8080/queue/pet

2. GET:
```
curl http://localhost:8080/queue/pet?timeout=5
```

# Запуск тестов:
```
go test -v
```
