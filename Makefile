DOCKER_REGISTRY ?= localhost:5000
PROJECT_NAME = web-chat
VERSION ?= latest

AUTH_IMAGE = $(DOCKER_REGISTRY)/auth:$(VERSION)
CHAT_IMAGE = $(DOCKER_REGISTRY)/chat:$(VERSION)
NOTIFICATION_IMAGE = $(DOCKER_REGISTRY)/notification:$(VERSION)

.PHONY: help build-auth build-chat build-all push-auth push-chat push-all docker-up docker-down docker-logs cleandoc test

# Show help
help:
	@echo "Доступные команды:"
	@echo "  build-auth     - Собрать Docker образ для auth-service"
	@echo "  build-chat     - Собрать Docker образ для chat-service"
	@echo "  build-all      - Собрать все Docker образы"
	@echo "  push-auth      - Отправить auth-service образ в registry"
	@echo "  push-chat      - Отправить chat-service образ в registry"
	@echo "  push-all       - Отправить все образы в registry"
	@echo "  docker-up      - Запустить все сервисы через Docker Compose"
	@echo "  docker-down    - Остановить все сервисы Docker Compose"
	@echo "  docker-logs    - Показать логи всех сервисов"
	@echo "  clean          - Очистить неиспользуемые Docker образы"
	@echo "  test           - Запустить тесты"

# Building Docker images
build-auth:
	@echo "Сборка auth-service образа..."
	docker build -t $(AUTH_IMAGE) ./auth-service

build-chat:
	@echo "Сборка chat-service образа..."
	docker build -t $(CHAT_IMAGE) ./chat-service

build-notification:
	@echo "Сборка chat-notification образа..."
	docker build -t $(NOTIFICATION_IMAGE) ./chat-notification

build-all: build-auth build-chat build-notification
	@echo "Все образы собраны успешно!"

# Push images to registry
push-auth: build-auth
	@echo "Отправка auth-service образа в registry..."
	docker push $(AUTH_IMAGE)

push-chat: build-chat
	@echo "Отправка chat-service образа в registry..."
	docker push $(CHAT_IMAGE)

push-notification: build-notification
	@echo "Отправка chat-service образа в registry..."
	docker push $(NOTIFICATION_IMAGE)

push-all: push-auth push-chat push-notification
	@echo "Все образы отправлены в registry!"

# Docker Compose commands
up:
	@echo "Запуск сервисов через Docker Compose..."
	docker-compose up -d
	@echo "Сервисы запущены! Веб-интерфейс доступен на http://localhost:8080"

down:
	@echo "Остановка сервисов Docker Compose..."
	docker-compose down

logs:
	@echo "Показ логов всех сервисов..."
	docker-compose logs -f

restart: docker-down docker-up

# Cleaning
clean:
	@echo "Очистка неиспользуемых Docker образов..."
	docker system prune -f
	docker image prune -f

clean-all:
	@echo "Полная очистка Docker (ВНИМАНИЕ: удалит все неиспользуемые данные)..."
	docker system prune -a -f

# Testing
test:
	@echo "Запуск тестов для auth-service..."
	cd auth-service && go test ./...
	@echo "Запуск тестов для chat-service..."
	cd chat-service && go test ./...

# Check services for ready
health-check:
	@echo "Проверка здоровья сервисов..."
	@curl -f http://localhost:8080 >/dev/null 2>&1 && echo "✅ Chat service: OK" || echo "❌ Chat service: FAIL"
	@grpc_health_probe -addr=localhost:50051 >/dev/null 2>&1 && echo "✅ Auth service: OK" || echo "❌ Auth service: FAIL"

# Resource monitoring
monitor:
	@echo "Мониторинг ресурсов Docker..."
	docker stats

# Backup data base
backup-db:
	@echo "Создание бэкапа базы данных..."
	docker exec web-chat-postgres pg_dump -U postgres webchat > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Бэкап создан!"

# Restore data base
restore-db:
	@read -p "Введите имя файла бэкапа: " backup_file; \
	docker exec -i web-chat-postgres psql -U postgres webchat < $$backup_file

# Deployment for dev
dev-setup: docker-up
	@echo "Ожидание готовности сервисов..."
	sleep 10
	@echo "Настройка завершена! Доступные URL:"
	@echo "  Веб-интерфейс: http://localhost:8080"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Auth gRPC: localhost:50051"

# Deployment for prod
prod-deploy: build-all push-all
	@echo "Продакшен развертывание завершено!"

# Update services
update-services: build-all
	@echo "Обновление сервисов..."
	docker-compose up -d --build
