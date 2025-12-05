.PHONY: help build up down restart logs clean dev-backend dev-frontend debug-backend test

# Variables
COMPOSE_FILE = docker-compose.yml
PROJECT_NAME = streammake

# Couleurs pour les messages
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
NC = \033[0m # No Color

help: ## Affiche cette aide
	@echo "$(GREEN)StreamMake - Commandes disponibles:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

build: ## Construit les images Docker
	@echo "$(GREEN)Construction des images Docker...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) build

up: ## Démarre tous les services
	@echo "$(GREEN)Démarrage de tous les services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d
	@echo "$(GREEN)Services démarrés!$(NC)"
	@echo "$(YELLOW)Frontend (via Nginx): http://localhost:3300$(NC)"
	@echo "$(YELLOW)Frontend (direct): http://localhost:3304$(NC)"
	@echo "$(YELLOW)Backend API: http://localhost:3301$(NC)"
	@echo "$(YELLOW)PostgreSQL: localhost:3302$(NC)"
	@echo "$(YELLOW)Redis: localhost:3303$(NC)"
	@echo "$(YELLOW)Debugger (Delve): localhost:2345$(NC)"

down: ## Arrête tous les services
	@echo "$(YELLOW)Arrêt de tous les services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down

restart: ## Redémarre tous les services
	@echo "$(YELLOW)Redémarrage des services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) restart

logs: ## Affiche les logs de tous les services
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f

logs-backend: ## Affiche les logs du backend uniquement
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f backend

logs-frontend: ## Affiche les logs du frontend uniquement
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f frontend

logs-db: ## Affiche les logs de la base de données
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f postgres

clean: ## Arrête et supprime les conteneurs, volumes et images
	@echo "$(RED)Suppression des conteneurs, volumes et images...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down -v --rmi all
	@echo "$(GREEN)Nettoyage terminé!$(NC)"

clean-volumes: ## Supprime uniquement les volumes
	@echo "$(YELLOW)Suppression des volumes...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down -v

dev-backend: ## Lance le backend en mode développement (sans Docker)
	@echo "$(GREEN)Lancement du backend en mode développement...$(NC)"
	cd backend && go run cmd/server/main.go

dev-frontend: ## Lance le frontend en mode développement (sans Docker)
	@echo "$(GREEN)Lancement du frontend en mode développement...$(NC)"
	cd frontend && npm run dev

debug-backend: ## Lance le backend avec Delve (debugger)
	@echo "$(GREEN)Lancement du backend en mode debug...$(NC)"
	@echo "$(YELLOW)Le debugger écoute sur le port 2345$(NC)"
	cd backend && dlv debug cmd/server/main.go --listen=:2345 --headless=true --api-version=2 --accept-multiclient

test: ## Lance les tests
	@echo "$(GREEN)Lancement des tests...$(NC)"
	cd backend && go test ./...

test-verbose: ## Lance les tests en mode verbeux
	@echo "$(GREEN)Lancement des tests en mode verbeux...$(NC)"
	cd backend && go test -v ./...

install-deps: ## Installe les dépendances
	@echo "$(GREEN)Installation des dépendances...$(NC)"
	cd backend && go mod download && go mod tidy
	cd frontend && npm install

migrate: ## Lance les migrations de base de données
	@echo "$(GREEN)Les migrations sont automatiques via GORM$(NC)"
	@echo "$(YELLOW)Les migrations SQL sont dans database/migrations/$(NC)"

shell-backend: ## Ouvre un shell dans le conteneur backend
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec backend sh

shell-db: ## Ouvre un shell PostgreSQL
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec postgres psql -U streammake -d streammake_db

status: ## Affiche le statut des services
	@echo "$(GREEN)Statut des services:$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) ps

rebuild: ## Reconstruit et redémarre les services
	@echo "$(GREEN)Reconstruction et redémarrage...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d --build

stop: ## Arrête les services sans les supprimer
	@echo "$(YELLOW)Arrêt des services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) stop

start: ## Démarre les services arrêtés
	@echo "$(GREEN)Démarrage des services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) start

