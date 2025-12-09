.PHONY: help build up down restart logs clean dev-backend dev-frontend debug-backend test install-deps status shell-backend shell-db test-api test-all test-upload test-comprehensive install-torrents

# Variables
PROJECT_NAME := streammake
DOCKER_COMPOSE_FILE := docker-compose.yml
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

help:
	@echo "${GREEN}Commandes disponibles:${NC}"
	@echo "${YELLOW}  build${NC}          Construire les images Docker"
	@echo "${YELLOW}  up${NC}             Démarrer tous les services en mode détaché"
	@echo "${YELLOW}  down${NC}           Arrêter et supprimer tous les services"
	@echo "${YELLOW}  restart${NC}        Redémarrer tous les services"
	@echo "${YELLOW}  logs${NC}           Afficher les logs de tous les services"
	@echo "${YELLOW}  logs-<service>${NC} Afficher les logs d'un service spécifique (ex: logs-backend)"
	@echo "${YELLOW}  clean${NC}          Nettoyer les conteneurs, volumes et réseaux arrêtés"
	@echo "${YELLOW}  dev-backend${NC}    Lancer le backend en mode développement (hors Docker)"
	@echo "${YELLOW}  dev-frontend${NC}  Lancer le frontend en mode développement (hors Docker)"
	@echo "${YELLOW}  debug-backend${NC}  Lancer le backend avec le debugger Delve"
	@echo "${YELLOW}  test${NC}           Exécuter les tests backend"
	@echo "${YELLOW}  test-api${NC}      Tester l'API (CORS, login, etc.)"
	@echo "${YELLOW}  test-all${NC}      Exécuter tous les tests complets de l'application"
	@echo "${YELLOW}  test-upload${NC}   Tester l'upload de vidéos"
	@echo "${YELLOW}  test-comprehensive${NC} Tests complets backend + frontend (toutes fonctionnalités)"
	@echo "${YELLOW}  install-deps${NC}  Installer les dépendances backend Go"
	@echo "${YELLOW}  status${NC}        Afficher le statut des services Docker"
	@echo "${YELLOW}  shell-backend${NC}  Accéder au shell du conteneur backend"
	@echo "${YELLOW}  shell-db${NC}       Accéder au shell PostgreSQL"
	@echo "${YELLOW}  install-torrents${NC} Installer et démarrer les clients torrent"

build:
	@echo "${GREEN}Construction des images Docker...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) build

up:
	@echo "${GREEN}Démarrage de tous les services...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d
	@echo "${GREEN}Attente du démarrage des services...${NC}"
	@sleep 5
	@echo "${GREEN}Exécution des tests API...${NC}"
	@./scripts/test-api.sh || true

down:
	@echo "${YELLOW}Arrêt de tous les services...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down

restart: down up

logs:
	@echo "${GREEN}Affichage des logs de tous les services (Ctrl+C pour quitter)...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs -f

logs-%:
	@echo "${GREEN}Affichage des logs du service $* (Ctrl+C pour quitter)...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs -f $*

clean:
	@echo "${YELLOW}Nettoyage des conteneurs, volumes et réseaux...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --volumes --rmi all --remove-orphans

dev-backend:
	@echo "${GREEN}Démarrage du backend en mode développement...${NC}"
	cd backend && go run ./cmd/server

dev-frontend:
	@echo "${GREEN}Démarrage du frontend en mode développement...${NC}"
	cd frontend && npm run dev

debug-backend:
	@echo "${GREEN}Démarrage du backend en mode debug avec Delve...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d backend
	docker exec -it $(PROJECT_NAME)-backend /usr/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/server

test:
	@echo "${GREEN}Exécution des tests backend...${NC}"
	cd backend && go test ./...

test-api:
	@echo "${GREEN}Test de l'API...${NC}"
	@./scripts/test-api.sh

test-all:
	@echo "${GREEN}Tests complets de l'application...${NC}"
	@./scripts/test-all.sh

test-upload:
	@echo "${GREEN}Test de l'upload de vidéos...${NC}"
	@./scripts/test-upload.sh

test-comprehensive:
	@echo "${GREEN}Tests complets backend + frontend...${NC}"
	@./scripts/test-comprehensive.sh

install-deps:
	@echo "${GREEN}Installation des dépendances Go...${NC}"
	cd backend && go mod tidy

status:
	@echo "${GREEN}Statut des services Docker:${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) ps
	@echo ""
	@echo "${GREEN}Test rapide de l'API:${NC}"
	@./scripts/test-api.sh || true

shell-backend:
	@echo "${GREEN}Accès au shell du conteneur backend...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) exec backend sh

shell-db:
	@echo "${GREEN}Accès au shell du conteneur PostgreSQL...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) exec postgres psql -U streammake streammake_db

install-torrents:
	@echo "${GREEN}Installation et démarrage des clients torrent...${NC}"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d transmission qbittorrent deluge rtorrent
	@echo "${GREEN}✅ Clients torrent démarrés:${NC}"
	@echo "${YELLOW}  - Transmission: http://localhost:3305${NC}"
	@echo "${YELLOW}  - qBittorrent: http://localhost:3306${NC}"
	@echo "${YELLOW}  - Deluge: http://localhost:3307${NC}"
	@echo "${YELLOW}  - ruTorrent: http://localhost:3308${NC}"
