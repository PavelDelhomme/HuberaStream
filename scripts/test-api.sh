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

