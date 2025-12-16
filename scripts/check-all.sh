#!/bin/bash

# Script de vérification complète de StreamMake
# Vérifie tous les éléments avant le démarrage de l'application
# Usage: make check ou ./scripts/check-all.sh

# Ne pas utiliser set -e pour permettre la continuation même en cas d'erreur
# set -e

# Couleurs
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Compteurs
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_WARNINGS=0

# Fonction pour afficher un résultat
check_result() {
    local status=$1
    local message=$2
    local details=$3
    
    if [ "$status" = "pass" ]; then
        echo -e "${GREEN}✓${NC} $message"
        ((CHECKS_PASSED++))
    elif [ "$status" = "fail" ]; then
        echo -e "${RED}✗${NC} $message"
        if [ -n "$details" ]; then
            echo -e "  ${YELLOW}→${NC} $details"
        fi
        ((CHECKS_FAILED++))
    elif [ "$status" = "warn" ]; then
        echo -e "${YELLOW}⚠${NC} $message"
        if [ -n "$details" ]; then
            echo -e "  ${YELLOW}→${NC} $details"
        fi
        ((CHECKS_WARNINGS++))
    fi
}

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  🔍 VÉRIFICATION COMPLÈTE DE STREAMMAKE${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# ============================================================================
# 1. VÉRIFICATION DES PRÉREQUIS
# ============================================================================
echo -e "${CYAN}[1/10] Vérification des prérequis...${NC}"

# Docker
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version | cut -d' ' -f3 | cut -d',' -f1)
    check_result "pass" "Docker installé (version $DOCKER_VERSION)"
else
    check_result "fail" "Docker non installé" "Installez Docker: https://docs.docker.com/get-docker/"
fi

# Docker Compose
if command -v docker-compose &> /dev/null || docker compose version &> /dev/null; then
    check_result "pass" "Docker Compose installé"
else
    check_result "fail" "Docker Compose non installé"
fi

# Go (pour compilation locale)
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | cut -d' ' -f3)
    check_result "pass" "Go installé ($GO_VERSION)"
else
    check_result "warn" "Go non installé" "Requis pour make rebuild-backend-local"
fi

# Node.js (pour compilation frontend)
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version)
    check_result "pass" "Node.js installé ($NODE_VERSION)"
else
    check_result "warn" "Node.js non installé" "Requis pour make dev-frontend"
fi

# FFmpeg
if command -v ffmpeg &> /dev/null; then
    FFMPEG_VERSION=$(ffmpeg -version | head -n1 | cut -d' ' -f3)
    check_result "pass" "FFmpeg installé (version $FFMPEG_VERSION)"
else
    check_result "warn" "FFmpeg non installé" "Requis pour le transcodage vidéo"
fi

echo ""

# ============================================================================
# 2. VÉRIFICATION DE LA STRUCTURE DES FICHIERS
# ============================================================================
echo -e "${CYAN}[2/10] Vérification de la structure des fichiers...${NC}"

# Fichiers essentiels
ESSENTIAL_FILES=(
    "docker-compose.yml"
    "Makefile"
    "backend/go.mod"
    "backend/go.sum"
    "backend/cmd/server/main.go"
    "frontend/package.json"
    "nginx/nginx.conf"
)

for file in "${ESSENTIAL_FILES[@]}"; do
    if [ -f "$file" ]; then
        check_result "pass" "Fichier présent: $file"
    else
        check_result "fail" "Fichier manquant: $file"
    fi
done

# Dossiers essentiels
ESSENTIAL_DIRS=(
    "backend"
    "frontend"
    "scripts"
    "nginx"
    "database/migrations"
)

for dir in "${ESSENTIAL_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        check_result "pass" "Dossier présent: $dir"
    else
        check_result "fail" "Dossier manquant: $dir"
    fi
done

echo ""

# ============================================================================
# 3. VÉRIFICATION DE LA COMPILATION BACKEND
# ============================================================================
echo -e "${CYAN}[3/10] Vérification de la compilation backend...${NC}"

if command -v go &> /dev/null; then
    cd backend
    
    # Vérifier go.mod
    if go mod verify &> /dev/null; then
        check_result "pass" "go.mod valide"
    else
        check_result "fail" "go.mod invalide" "Exécutez: cd backend && go mod tidy"
    fi
    
    # Vérifier les dépendances
    if go mod download &> /dev/null; then
        check_result "pass" "Dépendances Go téléchargées"
    else
        check_result "warn" "Erreur lors du téléchargement des dépendances"
    fi
    
    # Tentative de compilation (sans CGO pour test rapide)
    if CGO_ENABLED=0 go build -o /tmp/streammake-test ./cmd/server &> /dev/null; then
        check_result "pass" "Backend compile sans erreur"
        rm -f /tmp/streammake-test
    else
        check_result "fail" "Erreur de compilation backend" "Vérifiez les erreurs ci-dessus"
        # Afficher les erreurs
        CGO_ENABLED=0 go build ./cmd/server 2>&1 | head -20 || true
    fi
    
    # Vérifier les imports non utilisés
    if go vet ./... &> /dev/null; then
        check_result "pass" "go vet: aucun problème détecté"
    else
        check_result "warn" "go vet a détecté des problèmes" "Exécutez: cd backend && go vet ./..."
    fi
    
    cd ..
else
    check_result "warn" "Go non installé, compilation backend ignorée"
fi

echo ""

# ============================================================================
# 4. VÉRIFICATION DE LA COMPILATION FRONTEND
# ============================================================================
echo -e "${CYAN}[4/10] Vérification de la compilation frontend...${NC}"

if [ -f "frontend/package.json" ]; then
    cd frontend
    
    # Vérifier node_modules
    if [ -d "node_modules" ]; then
        check_result "pass" "node_modules présent"
    else
        check_result "warn" "node_modules absent" "Exécutez: cd frontend && npm install"
    fi
    
    # Vérifier package.json
    if npm list --depth=0 &> /dev/null; then
        check_result "pass" "package.json valide"
    else
        check_result "warn" "Erreur dans package.json" "Exécutez: cd frontend && npm install"
    fi
    
    # Test de build (si vite est disponible)
    if command -v npm &> /dev/null && npm list vite &> /dev/null; then
        if npm run build --if-present &> /dev/null; then
            check_result "pass" "Frontend compile sans erreur"
        else
            check_result "warn" "Erreur lors du build frontend" "Vérifiez: cd frontend && npm run build"
        fi
    else
        check_result "warn" "Vite non disponible, build frontend ignoré"
    fi
    
    cd ..
else
    check_result "fail" "frontend/package.json introuvable"
fi

echo ""

# ============================================================================
# 5. VÉRIFICATION DES VARIABLES D'ENVIRONNEMENT
# ============================================================================
echo -e "${CYAN}[5/10] Vérification des variables d'environnement...${NC}"

# Vérifier docker-compose.yml pour les variables essentielles
if grep -q "DB_HOST\|DB_USER\|DB_PASSWORD" docker-compose.yml 2>/dev/null; then
    check_result "pass" "Variables de base de données configurées dans docker-compose.yml"
else
    check_result "warn" "Variables de base de données non trouvées"
fi

if grep -q "JWT_SECRET" docker-compose.yml 2>/dev/null; then
    check_result "pass" "JWT_SECRET configuré"
else
    check_result "warn" "JWT_SECRET non configuré (utilise la valeur par défaut)"
fi

if grep -q "ADMIN_EMAIL\|ADMIN_PASSWORD" docker-compose.yml 2>/dev/null; then
    check_result "pass" "Variables admin configurées"
else
    check_result "warn" "Variables admin non configurées"
fi

echo ""

# ============================================================================
# 6. VÉRIFICATION DES SERVICES DOCKER
# ============================================================================
echo -e "${CYAN}[6/10] Vérification des services Docker...${NC}"

# Vérifier que docker-compose.yml est valide
if docker-compose config &> /dev/null; then
    check_result "pass" "docker-compose.yml valide"
else
    check_result "fail" "docker-compose.yml invalide" "Vérifiez la syntaxe YAML"
    docker-compose config 2>&1 | head -10
fi

# Vérifier les images requises
REQUIRED_IMAGES=(
    "postgres:15-alpine"
    "redis:7-alpine"
    "nginx:alpine"
)

for image in "${REQUIRED_IMAGES[@]}"; do
    if docker images "$image" --format "{{.Repository}}:{{.Tag}}" | grep -q "$image" 2>/dev/null || docker pull "$image" &> /dev/null; then
        check_result "pass" "Image disponible: $image"
    else
        check_result "warn" "Image non disponible: $image" "Sera téléchargée au premier démarrage"
    fi
done

# Vérifier les conteneurs en cours d'exécution
if docker-compose ps 2>/dev/null | grep -q "Up"; then
    RUNNING_CONTAINERS=$(docker-compose ps --services --filter "status=running" 2>/dev/null | wc -l)
    check_result "pass" "$RUNNING_CONTAINERS service(s) en cours d'exécution"
else
    check_result "warn" "Aucun service en cours d'exécution" "Démarrez avec: make up"
fi

echo ""

# ============================================================================
# 7. VÉRIFICATION DE LA BASE DE DONNÉES
# ============================================================================
echo -e "${CYAN}[7/10] Vérification de la base de données...${NC}"

# Vérifier les migrations
if [ -d "database/migrations" ] && [ "$(ls -A database/migrations/*.sql 2>/dev/null)" ]; then
    MIGRATION_COUNT=$(ls -1 database/migrations/*.sql 2>/dev/null | wc -l)
    check_result "pass" "$MIGRATION_COUNT migration(s) SQL trouvée(s)"
else
    check_result "warn" "Aucune migration SQL trouvée"
fi

# Tester la connexion à la base de données (si les services sont démarrés)
if docker-compose ps postgres 2>/dev/null | grep -q "Up"; then
    if docker-compose exec -T postgres pg_isready -U streammake -d streammake_db &> /dev/null; then
        check_result "pass" "PostgreSQL accessible"
        
        # Vérifier les tables
        TABLE_COUNT=$(docker-compose exec -T postgres psql -U streammake -d streammake_db -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ' || echo "0")
        if [ "$TABLE_COUNT" -gt 0 ]; then
            check_result "pass" "$TABLE_COUNT table(s) dans la base de données"
        else
            check_result "warn" "Aucune table dans la base de données" "Les migrations seront exécutées au démarrage"
        fi
    else
        check_result "warn" "PostgreSQL non accessible" "Vérifiez que le service est démarré"
    fi
else
    check_result "warn" "PostgreSQL non démarré" "Démarrez avec: make up"
fi

echo ""

# ============================================================================
# 8. VÉRIFICATION DES ROUTES API
# ============================================================================
echo -e "${CYAN}[8/10] Vérification des routes API...${NC}"

# Vérifier que le backend compile et a les routes
if [ -f "backend/cmd/server/main.go" ]; then
    # Vérifier que les handlers existent
    HANDLERS=(
        "backend/internal/api/auth_handler.go"
        "backend/internal/api/video_handler.go"
        "backend/internal/api/routes.go"
    )
    
    for handler in "${HANDLERS[@]}"; do
        if [ -f "$handler" ]; then
            check_result "pass" "Handler présent: $(basename $handler)"
        else
            check_result "fail" "Handler manquant: $handler"
        fi
    done
    
    # Vérifier que l'API est accessible (si les services sont démarrés)
    if docker-compose ps backend 2>/dev/null | grep -q "Up"; then
        sleep 2
        if curl -s http://localhost:3301/health &> /dev/null || curl -s http://localhost:3300/api/categories &> /dev/null; then
            check_result "pass" "API accessible"
        else
            check_result "warn" "API non accessible" "Vérifiez les logs: make logs-backend"
        fi
    fi
else
    check_result "fail" "backend/cmd/server/main.go introuvable"
fi

echo ""

# ============================================================================
# 9. VÉRIFICATION DES ERREURS DE CODE
# ============================================================================
echo -e "${CYAN}[9/10] Vérification des erreurs de code...${NC}"

# Backend: vérifier les erreurs de compilation
if command -v go &> /dev/null; then
    cd backend
    
    # Vérifier les erreurs de syntaxe (gofmt est toujours disponible)
    if gofmt -l . 2>/dev/null | grep -q .; then
        FORMATTED_FILES=$(gofmt -l . 2>/dev/null | wc -l)
        check_result "warn" "$FORMATTED_FILES fichier(s) nécessitent un formatage" "Exécutez: gofmt -w ."
    else
        check_result "pass" "Code Go correctement formaté"
    fi
    
    # Vérifier les imports non utilisés (si goimports est disponible)
    if command -v goimports &> /dev/null; then
        UNUSED_IMPORTS=$(goimports -l . 2>/dev/null | wc -l || echo "0")
        if [ "$UNUSED_IMPORTS" = "0" ]; then
            check_result "pass" "Aucun import non utilisé détecté"
        else
            check_result "warn" "$UNUSED_IMPORTS fichier(s) avec imports non utilisés" "Exécutez: goimports -w ."
        fi
    else
        check_result "warn" "goimports non installé" "Installez: go install golang.org/x/tools/cmd/goimports@latest"
    fi
    
    cd ..
fi

# Frontend: vérifier les erreurs de syntaxe (basique)
if [ -f "frontend/package.json" ] && command -v npm &> /dev/null; then
    cd frontend
    
    # Vérifier que les fichiers JSX/JS sont valides (basique)
    if find src -name "*.jsx" -o -name "*.js" 2>/dev/null | head -1 | grep -q .; then
        check_result "pass" "Fichiers frontend présents"
    else
        check_result "warn" "Aucun fichier JSX/JS trouvé dans src/"
    fi
    
    cd ..
fi

echo ""

# ============================================================================
# 10. VÉRIFICATION DES CONFIGURATIONS
# ============================================================================
echo -e "${CYAN}[10/10] Vérification des configurations...${NC}"

# Nginx
if [ -f "nginx/nginx.conf" ]; then
    if nginx -t -c "$(pwd)/nginx/nginx.conf" &> /dev/null || docker run --rm -v "$(pwd)/nginx:/etc/nginx:ro" nginx:alpine nginx -t &> /dev/null; then
        check_result "pass" "Configuration Nginx valide"
    else
        check_result "warn" "Configuration Nginx invalide" "Vérifiez nginx/nginx.conf"
    fi
else
    check_result "fail" "nginx/nginx.conf introuvable"
fi

# Docker Compose
if docker-compose config &> /dev/null; then
    check_result "pass" "Configuration Docker Compose valide"
else
    check_result "fail" "Configuration Docker Compose invalide"
fi

# Vérifier les ports
PORTS_IN_USE=()
for port in 3300 3301 3302 3303 3304; do
    if lsof -i :$port &> /dev/null || netstat -tuln 2>/dev/null | grep -q ":$port "; then
        PORTS_IN_USE+=($port)
    fi
done

if [ ${#PORTS_IN_USE[@]} -gt 0 ]; then
    check_result "warn" "Ports en cours d'utilisation: ${PORTS_IN_USE[*]}" "Assurez-vous que ce sont les services StreamMake"
else
    check_result "pass" "Ports disponibles"
fi

echo ""

# ============================================================================
# RÉSUMÉ
# ============================================================================
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  📊 RÉSUMÉ DE LA VÉRIFICATION${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${GREEN}✓ Vérifications réussies: $CHECKS_PASSED${NC}"
echo -e "${YELLOW}⚠ Avertissements: $CHECKS_WARNINGS${NC}"
echo -e "${RED}✗ Vérifications échouées: $CHECKS_FAILED${NC}"
echo ""

TOTAL_CHECKS=$((CHECKS_PASSED + CHECKS_FAILED + CHECKS_WARNINGS))
if [ $TOTAL_CHECKS -gt 0 ]; then
    SUCCESS_RATE=$((CHECKS_PASSED * 100 / TOTAL_CHECKS))
    echo -e "${CYAN}Taux de réussite: ${SUCCESS_RATE}%${NC}"
fi

echo ""

if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ Toutes les vérifications critiques sont passées !${NC}"
    echo -e "${GREEN}Vous pouvez démarrer l'application avec: make up${NC}"
    exit 0
else
    echo -e "${RED}❌ Certaines vérifications critiques ont échoué${NC}"
    echo -e "${YELLOW}Corrigez les erreurs avant de démarrer l'application${NC}"
    exit 1
fi

