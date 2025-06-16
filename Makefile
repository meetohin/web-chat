.PHONY: up down restart logs clean clean-all

up:
	@echo "Запуск сервисов через Docker Compose..."
	docker-compose up -d
	@echo "Сервисы запущены! Веб-интерфейс доступен на http://localhost:8080"

down:
	@echo "Остановка сервисов Docker Compose..."
	docker-compose down

restart: down up

logs:
	@echo "Показ логов всех сервисов..."
	docker-compose logs -f

clean:
	@echo "Очистка неиспользуемых Docker образов..."
	docker system prune -f
	docker image prune -f

clean-all:
	@echo "Полная очистка Docker..."
	docker system prune -a -f