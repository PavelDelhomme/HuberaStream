.PHONY: help build rebuild rebuild-backend rebuild-backend-local rebuild-backend-force up down down-force start stop stop-force run restart restart-backend logs logs-follow logs-% clean clean-all dev dev-backend dev-frontend debug-backend check test test-api test-metadata test-all install-deps status shell-backend shell-db install-torrents setup-images

# Variables
PROJECT_NAME := streammake
DOCKER_COMPOSE_FILE := docker-compose.yml
STREAMMAKE_CONTAINERS := streammake-postgres streammake-redis streammake-backend streammake-frontend streammake-nginx streammake-transmission streammake-qbittorrent streammake-deluge streammake-rtorrent
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

help:
	@echo "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	@echo "${GREEN}  STREAMMAKE - Commandes Make${NC}"
	@echo "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	@echo ""
	@echo "${YELLOW}📦 CONSTRUCTION:${NC}"
	@echo "  ${GREEN}build${NC}              Construire les images Docker"
	@echo "  ${GREEN}rebuild${NC}            Reconstruire toutes les images"
	@echo "  ${GREEN}rebuild-backend${NC}    Reconstruire uniquement le backend"
	@echo "  ${GREEN}rebuild-backend-local${NC} Compiler le backend localement et le monter"
	@echo ""
	@echo "${YELLOW}🚀 DÉMARRAGE / ARRÊT:${NC}"
	@echo "  ${GREEN}up / start / run${NC}   Démarrer tous les services"
	@echo "  ${GREEN}down / stop${NC}        Arrêter tous les services"
	@echo "  ${GREEN}down-force / stop-force${NC} Arrêter FORCÉMENT (kill)"
	@echo "  ${GREEN}restart${NC}            Redémarrer tous les services"
	@echo "  ${GREEN}restart-backend${NC}    Redémarrer uniquement le backend"
	@echo ""
	@echo "${YELLOW}🧹 NETTOYAGE:${NC}"
	@echo "  ${GREEN}clean${NC}              Nettoyer conteneurs, volumes, réseaux"
	@echo "  ${GREEN}clean-all${NC}          Nettoyer TOUT (⚠️  destructif, demande confirmation)"
	@echo ""
	@echo "${YELLOW}📋 LOGS:${NC}"
	@echo "  ${GREEN}logs${NC}               Afficher les logs (scrollable, recommandé)"
	@echo "  ${GREEN}logs-follow${NC}        Suivre les logs en temps réel"
	@echo "  ${GREEN}logs-<service>${NC}     Logs d'un service (ex: logs-backend)"
	@echo ""
	@echo "${YELLOW}💻 DÉVELOPPEMENT:${NC}"
	@echo "  ${GREEN}dev${NC}                Mode développement avec hot reload"
	@echo "  ${GREEN}dev-backend${NC}        Backend hors Docker (go run)"
	@echo "  ${GREEN}dev-frontend${NC}       Frontend hors Docker (npm run dev)"
	@echo "  ${GREEN}debug-backend${NC}      Backend avec debugger Delve"
	@echo ""
	@echo "${YELLOW}🧪 TESTS:${NC}"
	@echo "  ${GREEN}check${NC}             Vérification complète avant démarrage (recommandé)"
	@echo "  ${GREEN}test${NC}               Tests backend Go"
	@echo "  ${GREEN}test-api${NC}           Tests API"
	@echo "  ${GREEN}test-metadata${NC}       Test recherche métadonnées"
	@echo "  ${GREEN}test-all${NC}           Tests complets (API + fonctionnalités)"
	@echo ""
	@echo "${YELLOW}🔍 VÉRIFICATION CODE:${NC}"
	@echo "  ${GREEN}check-frontend${NC}     Vérifier erreurs frontend (ESLint, TypeScript)"
	@echo "  ${GREEN}check-backend${NC}      Vérifier erreurs backend (go vet, gofmt)"
	@echo "  ${GREEN}check-lint${NC}         Vérifier erreurs de linting (frontend + backend)"
	@echo "  ${GREEN}check-compile${NC}      Vérifier compilation (frontend + backend)"
	@echo "  ${GREEN}format-backend${NC}     Formater automatiquement le code backend (gofmt)"
	@echo ""
	@echo "${YELLOW}🔧 UTILITAIRES:${NC}"
	@echo "  ${GREEN}status${NC}             Statut des services"
	@echo "  ${GREEN}shell-backend${NC}      Shell du conteneur backend"
	@echo "  ${GREEN}shell-db${NC}           Shell PostgreSQL"
	@echo "  ${GREEN}install-deps${NC}       Installer dépendances Go"
	@echo "  ${GREEN}install-torrents${NC}  Démarrer les clients torrent"
	@echo "  ${GREEN}setup-images${NC}       Créer les images Docker (si buildx échoue)"
	@echo ""

# ============================================================================
# CONSTRUCTION
# ============================================================================

build:
	@echo "${GREEN}📦 Construction des images Docker...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) build

rebuild: build
	@echo "${GREEN}✅ Images reconstruites${NC}"

rebuild-backend:
	@echo "${GREEN}🔨 Reconstruction du backend...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) build backend
	@echo "${GREEN}✅ Backend reconstruit${NC}"

rebuild-backend-local:
	@echo "${GREEN}🔨 Compilation locale du backend...${NC}"
	@./scripts/rebuild-backend-local.sh

rebuild-backend-force:
	@echo "${YELLOW}⚠️  Reconstruction FORCÉE du backend...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) build --no-cache backend
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) restart backend
	@echo "${GREEN}✅ Backend reconstruit et redémarré${NC}"

# ============================================================================
# DÉMARRAGE / ARRÊT
# ============================================================================

up:
	@echo "${GREEN}🚀 Démarrage/Redémarrage de tous les services...${NC}"
	@echo "${YELLOW}💡 Vérification des images Docker...${NC}"
	@if ! docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "streammake-backend:latest" 2>/dev/null; then \
		echo "${YELLOW}⚠️  Image backend manquante. Utilisation de la compilation locale...${NC}"; \
		make rebuild-backend-local || { \
			echo "${YELLOW}⚠️  Compilation locale échouée. Création d'une image minimale...${NC}"; \
			mkdir -p .backend-bin && \
			if [ ! -f .backend-bin/main ]; then \
				cd backend && CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-s -w' -o ../.backend-bin/main ./cmd/server && cd ..; \
			fi && \
			if [ -f .backend-bin/main ]; then \
				echo "FROM alpine:latest" > .backend-bin/Dockerfile && \
				echo "RUN apk add --no-cache ca-certificates tzdata" >> .backend-bin/Dockerfile && \
				echo "COPY main /app/main" >> .backend-bin/Dockerfile && \
				echo "RUN chmod +x /app/main" >> .backend-bin/Dockerfile && \
				echo "WORKDIR /app" >> .backend-bin/Dockerfile && \
				echo "CMD [\"/app/main\"]" >> .backend-bin/Dockerfile && \
				cd .backend-bin && docker build -t streammake-backend:latest . && cd .. && \
				echo "${GREEN}✅ Image backend minimale créée${NC}"; \
			else \
				echo "${RED}❌ Impossible de créer l'image backend${NC}"; \
				echo "${YELLOW}💡 Construisez manuellement avec: make rebuild-backend-local${NC}"; \
				exit 1; \
			fi; \
		}; \
	fi
	@if ! docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "streammake-frontend:latest" 2>/dev/null; then \
		echo "${YELLOW}⚠️  Image frontend manquante.${NC}"; \
		echo "${YELLOW}💡 Utilisez 'make setup-images' pour créer toutes les images${NC}"; \
		echo "${YELLOW}⚠️  Continuons sans image frontend personnalisée (utilisera node:20-alpine)${NC}"; \
	fi
	@echo "${GREEN}✅ Images vérifiées${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d --no-build || { \
			echo "${RED}❌ Erreur lors du démarrage des services${NC}"; \
			echo "${YELLOW}💡 Vérifiez les logs avec: make logs${NC}"; \
			exit 1; \
		}; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d --no-build || { \
			echo "${RED}❌ Erreur lors du démarrage des services${NC}"; \
			echo "${YELLOW}💡 Vérifiez les logs avec: make logs${NC}"; \
			exit 1; \
		}; \
	fi
	@sleep 5
	@echo "${GREEN}✅ Services démarrés/redémarrés${NC}"
	@echo "${YELLOW}🔍 Vérification de santé des services...${NC}"
	@echo "${YELLOW}   Attente que le backend soit prêt...${NC}"
	@MAX_RETRIES=30; RETRY=0; \
	while [ $$RETRY -lt $$MAX_RETRIES ]; do \
		if curl -s -f http://localhost:3300/api/categories > /dev/null 2>&1 || curl -s -f http://localhost:3301/api/categories > /dev/null 2>&1; then \
			echo "${GREEN}   ✅ Backend accessible${NC}"; \
			break; \
		fi; \
		RETRY=$$((RETRY + 1)); \
		if [ $$RETRY -lt $$MAX_RETRIES ]; then \
			sleep 1; \
		fi; \
	done; \
	if [ $$RETRY -eq $$MAX_RETRIES ]; then \
		echo "${RED}   ❌ Backend non accessible après $$MAX_RETRIES tentatives${NC}"; \
		echo "${YELLOW}   💡 Vérifiez les logs avec: make logs backend${NC}"; \
	fi
	@echo "${YELLOW}   Test de la route /api/users/me (doit retourner 401, pas 404)...${NC}"
	@STATUS=$$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3300/api/users/me -H "Authorization: Bearer invalid" 2>/dev/null || echo "000"); \
	if [ "$$STATUS" = "401" ] || [ "$$STATUS" = "403" ]; then \
		echo "${GREEN}   ✅ Route /api/users/me accessible ($$STATUS - normal avec token invalide)${NC}"; \
	elif [ "$$STATUS" = "404" ]; then \
		echo "${RED}   ❌ Route /api/users/me non trouvée (404)${NC}"; \
		echo "${YELLOW}   💡 Le backend doit être redémarré ou recompilé${NC}"; \
		echo "${YELLOW}   💡 Essayez: make restart-backend${NC}"; \
	else \
		echo "${YELLOW}   ⚠️  Route /api/users/me retourne $$STATUS (attendu: 401/403)${NC}"; \
	fi
	@echo "${YELLOW}Test API complet...${NC}"
	@./scripts/test-api.sh || true

down:
	@echo "${YELLOW}🛑 Arrêt de tous les services...${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true; \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --remove-orphans || true; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true; \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --remove-orphans || true; \
	fi
	@echo "${GREEN}✅ Services arrêtés${NC}"

down-force:
	@echo "${RED}⚠️  Arrêt FORCÉ de tous les services...${NC}"
	@for container in $(STREAMMAKE_CONTAINERS); do \
		docker kill $$container 2>/dev/null || true; \
	done
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) kill || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --remove-orphans || true
	@echo "${GREEN}✅ Services arrêtés (forcé)${NC}"

start: up
run: up
stop: down
stop-force: down-force

restart:
	@echo "${GREEN}🔄 Redémarrage de tous les services...${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) restart || { \
			echo "${YELLOW}⚠️  Redémarrage échoué, tentative de stop/start...${NC}"; \
			docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true; \
			docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d --no-build; \
		}; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) restart || { \
			echo "${YELLOW}⚠️  Redémarrage échoué, tentative de stop/start...${NC}"; \
			docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true; \
			docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d --no-build; \
		}; \
	fi
	@sleep 5
	@echo "${GREEN}✅ Services redémarrés${NC}"

restart-backend:
	@echo "${GREEN}🔄 Redémarrage du backend...${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) restart backend || { \
			echo "${YELLOW}⚠️  Redémarrage échoué, tentative de stop/start...${NC}"; \
			docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop backend || true; \
			docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d --no-build backend; \
		}; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) restart backend || { \
			echo "${YELLOW}⚠️  Redémarrage échoué, tentative de stop/start...${NC}"; \
			docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop backend || true; \
			docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d --no-build backend; \
		}; \
	fi
	@sleep 5
	@echo "${GREEN}✅ Backend redémarré${NC}"
	@echo "${YELLOW}🔍 Vérification de santé du backend...${NC}"
	@MAX_RETRIES=30; RETRY=0; \
	while [ $$RETRY -lt $$MAX_RETRIES ]; do \
		if curl -s -f http://localhost:3300/api/categories > /dev/null 2>&1 || curl -s -f http://localhost:3301/api/categories > /dev/null 2>&1; then \
			echo "${GREEN}   ✅ Backend accessible${NC}"; \
			break; \
		fi; \
		RETRY=$$((RETRY + 1)); \
		if [ $$RETRY -lt $$MAX_RETRIES ]; then \
			sleep 1; \
		fi; \
	done; \
	if [ $$RETRY -eq $$MAX_RETRIES ]; then \
		echo "${RED}   ❌ Backend non accessible après $$MAX_RETRIES tentatives${NC}"; \
		echo "${YELLOW}   💡 Vérifiez les logs avec: make logs backend${NC}"; \
	fi
	@echo "${YELLOW}   Test de la route /api/users/me (doit retourner 401, pas 404)...${NC}"
	@STATUS=$$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3300/api/users/me -H "Authorization: Bearer invalid" 2>/dev/null || echo "000"); \
	if [ "$$STATUS" = "401" ] || [ "$$STATUS" = "403" ]; then \
		echo "${GREEN}   ✅ Route /api/users/me accessible ($$STATUS - normal avec token invalide)${NC}"; \
	elif [ "$$STATUS" = "404" ]; then \
		echo "${RED}   ❌ Route /api/users/me non trouvée (404)${NC}"; \
		echo "${YELLOW}   💡 Le backend doit être recompilé${NC}"; \
		echo "${YELLOW}   💡 Essayez: make rebuild-backend-local${NC}"; \
	else \
		echo "${YELLOW}   ⚠️  Route /api/users/me retourne $$STATUS (attendu: 401/403)${NC}"; \
	fi

# ============================================================================
# NETTOYAGE
# ============================================================================

clean:
	@echo "${YELLOW}🧹 Nettoyage des conteneurs, volumes et réseaux...${NC}"
	@for container in $(STREAMMAKE_CONTAINERS); do \
		docker kill $$container 2>/dev/null || true; \
	done
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) kill || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --volumes --remove-orphans || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --rmi local || true
	@echo "${GREEN}✅ Nettoyage terminé${NC}"

clean-all:
	@echo "${RED}⚠️  NETTOYAGE COMPLET - Suppression de TOUT${NC}"
	@echo "${YELLOW}Cette commande va supprimer même les images utilisées ailleurs !${NC}"
	@read -p "Êtes-vous sûr ? (oui/non): " confirm && [ "$$confirm" = "oui" ] || exit 1
	@for container in $(STREAMMAKE_CONTAINERS); do \
		docker kill $$container 2>/dev/null || true; \
	done
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) kill || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) stop || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --volumes --remove-orphans || true
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) down --rmi all --remove-orphans || true
	@echo "${GREEN}✅ Nettoyage complet terminé${NC}"

# ============================================================================
# LOGS
# ============================================================================

logs:
	@echo "${GREEN}📋 Logs StreamMake (scrollable, appuyez 'F' pour suivre, 'q' pour quitter)...${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=500 2>&1 | grep -E "(streammake|backend|frontend|nginx|postgres|redis)" | less -R +F || \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=500 2>&1 | less -R +F; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=500 2>&1 | grep -E "(streammake|backend|frontend|nginx|postgres|redis)" | less -R +F || \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=500 2>&1 | less -R +F; \
	fi

logs-follow:
	@echo "${GREEN}📺 Logs en temps réel StreamMake (Ctrl+C pour quitter)...${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=100 -f 2>&1 | grep -E "(streammake|backend|frontend|nginx|postgres|redis)" || \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=100 -f; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=100 -f 2>&1 | grep -E "(streammake|backend|frontend|nginx|postgres|redis)" || \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=100 -f; \
	fi

logs-%:
	@echo "${GREEN}📋 Logs du service $* (Ctrl+C pour quitter)...${NC}"
	@if command -v docker &> /dev/null && docker compose version &> /dev/null; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=200 -f $*; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) logs --tail=200 -f $*; \
	fi

# ============================================================================
# DÉVELOPPEMENT
# ============================================================================

dev:
	@echo "${GREEN}🚀 Mode développement avec hot reload...${NC}"
	@echo "${YELLOW}Frontend: http://localhost:3304${NC}"
	@echo "${YELLOW}Backend: http://localhost:3301${NC}"
	@echo "${YELLOW}Appuyez sur Ctrl+C pour arrêter${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up --build

dev-backend:
	@echo "${GREEN}💻 Backend en mode développement (hors Docker)...${NC}"
	@cd backend && go run ./cmd/server

dev-frontend:
	@echo "${GREEN}💻 Frontend en mode développement (hors Docker)...${NC}"
	@cd frontend && npm run dev

debug-backend:
	@echo "${GREEN}🐛 Backend avec debugger Delve...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d backend
	@docker exec -it $(PROJECT_NAME)-backend /usr/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/server

# ============================================================================
# TESTS
# ============================================================================

check:
	@echo "${GREEN}🔍 Vérification complète de StreamMake...${NC}"
	@./scripts/check-all.sh

test:
	@echo "${GREEN}🧪 Tests backend...${NC}"
	@cd backend && go test ./...

test-api:
	@echo "${GREEN}🧪 Tests API...${NC}"
	@echo "${YELLOW}💡 Les services seront démarrés automatiquement si nécessaire${NC}"
	@./scripts/test-api.sh || exit 1

test-metadata:
	@echo "${GREEN}🧪 Test recherche métadonnées...${NC}"
	@echo "${YELLOW}💡 Les services seront démarrés automatiquement si nécessaire${NC}"
	@./scripts/test-metadata.sh || exit 1

test-all:
	@echo "${GREEN}🧪 Tests complets...${NC}"
	@echo "${YELLOW}💡 Les services seront démarrés automatiquement si nécessaire${NC}"
	@./scripts/test-all.sh || exit 1

# ============================================================================
# UTILITAIRES
# ============================================================================

status:
	@echo "${GREEN}📊 Statut des services:${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) ps
	@echo ""
	@echo "${GREEN}Test API:${NC}"
	@./scripts/test-api.sh || true

shell-backend:
	@echo "${GREEN}🐚 Shell du conteneur backend...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) exec backend sh

shell-db:
	@echo "${GREEN}🐚 Shell PostgreSQL...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) exec postgres psql -U streammake streammake_db

# ============================================================================
# VÉRIFICATION CODE
# ============================================================================

check-frontend:
	@echo "${GREEN}🔍 Vérification du code frontend...${NC}"
	@if docker ps --format "{{.Names}}" | grep -q "^streammake-frontend$$"; then \
		echo "${YELLOW}🐳 Utilisation du conteneur Docker...${NC}"; \
		docker exec streammake-frontend sh -c "cd /app && npm run build --dry-run 2>&1 || echo 'Build check'" || true; \
		echo "${GREEN}✅ Vérification frontend terminée (conteneur)${NC}"; \
	elif command -v npm >/dev/null 2>&1; then \
		cd frontend && \
		echo "${YELLOW}📦 Installation des dépendances si nécessaire...${NC}"; \
		if [ ! -d node_modules ]; then \
			npm install --silent; \
		fi; \
		if [ -f package.json ]; then \
			echo "${YELLOW}🔍 Vérification des scripts disponibles...${NC}"; \
			if grep -q '"lint"' package.json; then \
				echo "${YELLOW}🔍 Vérification ESLint...${NC}"; \
				npm run lint 2>&1 || echo "${RED}⚠️  ESLint a trouvé des erreurs${NC}"; \
			else \
				echo "${YELLOW}⚠️  Script 'lint' non trouvé dans package.json${NC}"; \
			fi; \
			if [ -f tsconfig.json ] || find src -name "*.ts" -o -name "*.tsx" 2>/dev/null | head -1 | grep -q .; then \
				echo "${YELLOW}🔍 Vérification TypeScript...${NC}"; \
				if [ -d node_modules/typescript ]; then \
					npx --yes typescript tsc --noEmit 2>&1 || echo "${RED}⚠️  TypeScript a trouvé des erreurs${NC}"; \
				else \
					echo "${YELLOW}⚠️  TypeScript non installé, vérification ignorée${NC}"; \
				fi; \
			else \
				echo "${YELLOW}ℹ️  Projet JavaScript (pas TypeScript)${NC}"; \
			fi; \
		fi; \
		echo "${GREEN}✅ Vérification frontend terminée${NC}"; \
	else \
		echo "${RED}❌ npm n'est pas installé et conteneur non disponible${NC}"; \
		exit 1; \
	fi

check-backend:
	@echo "${GREEN}🔍 Vérification du code backend...${NC}"
	@if docker ps --format "{{.Names}}" | grep -q "^streammake-backend$$"; then \
		echo "${YELLOW}🐳 Utilisation du conteneur Docker...${NC}"; \
		echo "${YELLOW}🔍 Vérification go vet...${NC}"; \
		docker exec streammake-backend sh -c "cd /app && go vet ./... 2>&1" || echo "${RED}⚠️  go vet a trouvé des problèmes${NC}"; \
		echo "${YELLOW}🔍 Vérification gofmt...${NC}"; \
		UNFORMATTED=$$(docker exec streammake-backend sh -c "cd /app && gofmt -l . 2>/dev/null" | head -20); \
		if [ -n "$$UNFORMATTED" ]; then \
			echo "${RED}⚠️  Fichiers non formatés (premiers 20):${NC}"; \
			echo "$$UNFORMATTED" | sed 's/^/  /'; \
			echo "${YELLOW}💡 Exécutez 'make format-backend' pour corriger automatiquement${NC}"; \
		else \
			echo "${GREEN}✅ Tous les fichiers sont formatés${NC}"; \
		fi; \
		echo "${YELLOW}🔍 Vérification compilation...${NC}"; \
		docker exec streammake-backend sh -c "cd /app && go build ./... 2>&1" || echo "${RED}⚠️  Erreurs de compilation${NC}"; \
		echo "${GREEN}✅ Vérification backend terminée (conteneur)${NC}"; \
	elif command -v go >/dev/null 2>&1; then \
		cd backend && \
		echo "${YELLOW}🔍 Vérification go vet...${NC}"; \
		go vet ./... 2>&1 || echo "${RED}⚠️  go vet a trouvé des problèmes${NC}"; \
		echo "${YELLOW}🔍 Vérification gofmt...${NC}"; \
		UNFORMATTED=$$(gofmt -l . 2>/dev/null | head -20); \
		if [ -n "$$UNFORMATTED" ]; then \
			echo "${RED}⚠️  Fichiers non formatés (premiers 20):${NC}"; \
			echo "$$UNFORMATTED" | sed 's/^/  /'; \
			echo "${YELLOW}💡 Exécutez 'make format-backend' pour corriger automatiquement${NC}"; \
		else \
			echo "${GREEN}✅ Tous les fichiers sont formatés${NC}"; \
		fi; \
		echo "${YELLOW}🔍 Vérification compilation...${NC}"; \
		go build ./... 2>&1 || echo "${RED}⚠️  Erreurs de compilation${NC}"; \
		echo "${GREEN}✅ Vérification backend terminée${NC}"; \
	else \
		echo "${RED}❌ Go n'est pas installé et conteneur non disponible${NC}"; \
		exit 1; \
	fi

format-backend:
	@echo "${GREEN}🔧 Formatage du code backend avec gofmt...${NC}"
	@if command -v go >/dev/null 2>&1; then \
		cd backend && \
		gofmt -w . && \
		echo "${GREEN}✅ Code backend formaté${NC}"; \
	else \
		echo "${RED}❌ Go n'est pas installé${NC}"; \
		exit 1; \
	fi

check-lint: check-frontend check-backend
	@echo "${GREEN}✅ Vérification complète terminée${NC}"

check-compile:
	@echo "${GREEN}🔍 Vérification compilation frontend et backend...${NC}"
	@echo "${YELLOW}📦 Compilation frontend...${NC}"
	@if docker ps --format "{{.Names}}" | grep -q "^streammake-frontend$$"; then \
		echo "${YELLOW}🐳 Utilisation du conteneur Docker...${NC}"; \
		docker exec streammake-frontend sh -c "cd /app && npm run build 2>&1" | tail -30 || echo "${RED}⚠️  Erreur de compilation frontend${NC}"; \
	elif command -v npm >/dev/null 2>&1; then \
		cd frontend && \
		if [ ! -d node_modules ]; then \
			echo "${YELLOW}📦 Installation des dépendances...${NC}"; \
			npm install --silent; \
		fi; \
		if grep -q '"build"' package.json; then \
			npm run build 2>&1 | tail -30 || echo "${RED}⚠️  Erreur de compilation frontend${NC}"; \
		else \
			echo "${YELLOW}⚠️  Script 'build' non trouvé dans package.json${NC}"; \
		fi; \
	else \
		echo "${RED}❌ npm n'est pas installé et conteneur non disponible${NC}"; \
	fi
	@echo "${YELLOW}📦 Compilation backend...${NC}"
	@if docker ps --format "{{.Names}}" | grep -q "^streammake-backend$$"; then \
		echo "${YELLOW}🐳 Utilisation du conteneur Docker...${NC}"; \
		docker exec streammake-backend sh -c "cd /app && go build ./cmd/server 2>&1" || echo "${RED}⚠️  Erreur de compilation backend${NC}"; \
	elif command -v go >/dev/null 2>&1; then \
		cd backend && \
		go build ./cmd/server 2>&1 || echo "${RED}⚠️  Erreur de compilation backend${NC}"; \
	else \
		echo "${RED}❌ Go n'est pas installé et conteneur non disponible${NC}"; \
	fi
	@echo "${GREEN}✅ Vérification compilation terminée${NC}"

install-deps:
	@echo "${GREEN}📦 Installation des dépendances Go...${NC}"
	@cd backend && go mod tidy

install-torrents:
	@echo "${GREEN}📥 Démarrage des clients torrent...${NC}"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) -p $(PROJECT_NAME) up -d transmission qbittorrent deluge rtorrent
	@echo "${GREEN}✅ Clients torrent démarrés:${NC}"
	@echo "${YELLOW}  - Transmission: http://localhost:3305${NC}"
	@echo "${YELLOW}  - qBittorrent: http://localhost:3306${NC}"
	@echo "${YELLOW}  - Deluge: http://localhost:3307${NC}"
	@echo "${YELLOW}  - ruTorrent: http://localhost:3308${NC}"

setup-images:
	@echo "${GREEN}🔨 Création des images Docker...${NC}"
	@./scripts/setup-images.sh
