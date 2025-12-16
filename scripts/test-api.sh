#!/bin/bash

# Script de test pour vérifier que l'API fonctionne correctement

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Test de l'API StreamMake"
echo "=========================================="
echo ""

# Vérifier et démarrer les services si nécessaire
# Vérifier si le backend existe (même s'il redémarre)
BACKEND_EXISTS=$(docker compose ps 2>/dev/null | grep -c "streammake-backend" || docker-compose ps 2>/dev/null | grep -c "streammake-backend" || echo "0")
if [ "$BACKEND_EXISTS" = "0" ] || (! docker compose ps 2>/dev/null | grep -q "streammake-backend.*Up\|streammake-backend.*Restarting" && ! docker-compose ps 2>/dev/null | grep -q "streammake-backend.*Up\|streammake-backend.*Restarting"); then
    echo -e "${YELLOW}⚠️  Les services Docker ne sont pas démarrés. Démarrage...${NC}"
    echo -e "${YELLOW}💡 Utilisation de 'make up' pour construire et démarrer les services...${NC}"
    
    # Utiliser make up qui gère mieux le build
    cd "$(dirname "$0")/.." || exit 1
    if make up 2>&1; then
        echo -e "${GREEN}✅ Services démarrés${NC}"
    else
        echo -e "${RED}❌ Impossible de démarrer les services automatiquement${NC}"
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${YELLOW}💡 SOLUTION:${NC}"
        echo -e "${YELLOW}   1. Construisez le backend localement:${NC}"
        echo -e "${GREEN}      make rebuild-backend-local${NC}"
        echo -e "${YELLOW}   2. Créez une image Docker minimale:${NC}"
        echo -e "${GREEN}      cd .backend-bin && docker build -t streammake-backend:latest . && cd ..${NC}"
        echo -e "${YELLOW}   3. Démarrez les services:${NC}"
        echo -e "${GREEN}      docker compose up -d --no-build${NC}"
        echo -e "${YELLOW}   4. Relancez les tests:${NC}"
        echo -e "${GREEN}      make test-api${NC}"
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        exit 1
    fi
    cd - > /dev/null || exit 1
    
    echo -e "${YELLOW}⏳ Attente du démarrage des services (20 secondes)...${NC}"
    sleep 20
    
    # Vérifier que l'API répond
    MAX_RETRIES=10
    RETRY_COUNT=0
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -s http://localhost:3300/api/categories &> /dev/null || curl -s http://localhost:3301/health &> /dev/null; then
            echo -e "${GREEN}✅ Services démarrés${NC}"
            echo ""
            break
        fi
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
            sleep 2
        fi
    done
    
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Impossible de démarrer les services${NC}"
        echo -e "${YELLOW}💡 Démarrez manuellement avec: make up${NC}"
        exit 1
    fi
fi

# Test 1: OPTIONS preflight
echo -e "${YELLOW}Test 1: Requête OPTIONS (preflight CORS)...${NC}"
OPTIONS_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type")

if [ "$OPTIONS_RESPONSE" = "204" ]; then
  echo -e "${GREEN}✓ OPTIONS retourne 204 (No Content)${NC}"
else
  echo -e "${RED}✗ OPTIONS retourne $OPTIONS_RESPONSE (attendu: 204)${NC}"
fi
echo ""

# Test 2: POST login
echo -e "${YELLOW}Test 2: Requête POST /api/auth/login...${NC}"
LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}')

HTTP_CODE=$(echo "$LOGIN_RESPONSE" | tail -n 1)
BODY=$(echo "$LOGIN_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
  echo -e "${GREEN}✓ Login retourne 200${NC}"
  if echo "$BODY" | grep -q "token"; then
    echo -e "${GREEN}✓ Token présent dans la réponse${NC}"
  else
    echo -e "${YELLOW}⚠ Token non trouvé dans la réponse${NC}"
  fi
else
  echo -e "${RED}✗ Login retourne $HTTP_CODE (attendu: 200)${NC}"
  echo "Réponse: $BODY"
fi
echo ""

# Test 3: Vérifier les headers CORS
echo -e "${YELLOW}Test 3: Vérification des headers CORS...${NC}"
CORS_HEADERS=$(curl -s -D - -X POST http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Content-Type: application/json" \
  -d '{"email":"test","password":"test"}' -o /dev/null)

if echo "$CORS_HEADERS" | grep -qi "Access-Control-Allow-Origin"; then
  ORIGIN_HEADER=$(echo "$CORS_HEADERS" | grep -i "Access-Control-Allow-Origin" | head -1)
  echo -e "${GREEN}✓ Header Access-Control-Allow-Origin présent: $ORIGIN_HEADER${NC}"
else
  echo -e "${RED}✗ Header Access-Control-Allow-Origin manquant${NC}"
fi

if echo "$CORS_HEADERS" | grep -qi "Access-Control-Allow-Credentials"; then
  CREDENTIALS_HEADER=$(echo "$CORS_HEADERS" | grep -i "Access-Control-Allow-Credentials" | head -1)
  echo -e "${GREEN}✓ Header Access-Control-Allow-Credentials présent: $CREDENTIALS_HEADER${NC}"
else
  echo -e "${RED}✗ Header Access-Control-Allow-Credentials manquant${NC}"
fi
echo ""

# Test 4: Vérifier que les services sont en cours d'exécution
echo -e "${YELLOW}Test 4: Vérification des services Docker...${NC}"
if docker-compose ps | grep -q "streammake-backend.*Up"; then
  echo -e "${GREEN}✓ Backend est en cours d'exécution${NC}"
else
  echo -e "${RED}✗ Backend n'est pas en cours d'exécution${NC}"
fi

if docker-compose ps | grep -q "streammake-frontend.*Up"; then
  echo -e "${GREEN}✓ Frontend est en cours d'exécution${NC}"
else
  echo -e "${RED}✗ Frontend n'est pas en cours d'exécution${NC}"
fi

if docker-compose ps | grep -q "streammake-nginx.*Up"; then
  echo -e "${GREEN}✓ Nginx est en cours d'exécution${NC}"
else
  echo -e "${RED}✗ Nginx n'est pas en cours d'exécution${NC}"
fi
echo ""

echo "=========================================="
echo "Tests terminés"
echo "=========================================="

