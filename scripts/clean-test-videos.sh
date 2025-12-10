#!/bin/bash

# Script pour nettoyer les vidéos de test (titre commençant par "Test Video")

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_BASE_URL="http://localhost:3300/api"
ADMIN_TOKEN=""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  NETTOYAGE DES VIDÉOS DE TEST${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Connexion admin
echo -e "${BLUE}[1/3] Connexion admin...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}')

if [ "$(echo "$LOGIN_RESPONSE" | jq -r .token 2>/dev/null)" != "null" ]; then
  ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r .token)
  echo -e "${GREEN}✓ Connexion admin réussie${NC}"
else
  echo -e "${RED}✗ Connexion admin échouée${NC}"
  exit 1
fi
echo ""

# Récupérer la liste des vidéos de test
echo -e "${BLUE}[2/3] Récupération des vidéos de test...${NC}"
VIDEOS_RESPONSE=$(curl -s -X GET "$API_BASE_URL/videos?status=all&limit=1000" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

# Filtrer les vidéos dont le titre commence par "Test Video"
TEST_VIDEOS=$(echo "$VIDEOS_RESPONSE" | jq -r '.videos[] | select(.title | startswith("Test Video")) | .id' 2>/dev/null)

if [ -z "$TEST_VIDEOS" ]; then
  echo -e "${YELLOW}⚠ Aucune vidéo de test trouvée${NC}"
  echo ""
  echo -e "${BLUE}========================================${NC}"
  echo -e "${BLUE}  RÉSUMÉ${NC}"
  echo -e "${BLUE}========================================${NC}"
  echo ""
  echo -e "${GREEN}✓ Aucune vidéo de test à nettoyer${NC}"
  exit 0
fi

TEST_VIDEO_COUNT=$(echo "$TEST_VIDEOS" | wc -l)
echo -e "${GREEN}✓ $TEST_VIDEO_COUNT vidéo(s) de test trouvée(s)${NC}"
echo ""

# Supprimer les vidéos de test
echo -e "${BLUE}[3/3] Suppression des vidéos de test...${NC}"
DELETED_COUNT=0
FAILED_COUNT=0

while IFS= read -r VIDEO_ID; do
  if [ -n "$VIDEO_ID" ] && [ "$VIDEO_ID" != "null" ]; then
    DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE_URL/videos/$VIDEO_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN")
    
    HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n 1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
      echo -e "${GREEN}  ✓ Vidéo $VIDEO_ID supprimée${NC}"
      DELETED_COUNT=$((DELETED_COUNT + 1))
    else
      echo -e "${RED}  ✗ Erreur lors de la suppression de $VIDEO_ID (HTTP $HTTP_CODE)${NC}"
      FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
  fi
done <<< "$TEST_VIDEOS"

echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  RÉSUMÉ${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✓ $DELETED_COUNT vidéo(s) supprimée(s)${NC}"
if [ "$FAILED_COUNT" -gt 0 ]; then
  echo -e "${RED}✗ $FAILED_COUNT vidéo(s) n'a(ont) pas pu être supprimée(s)${NC}"
fi
echo ""

