#!/bin/bash

# Script de test pour l'upload de vidéos

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_BASE_URL="http://localhost:3300/api"
ADMIN_TOKEN=""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  TEST UPLOAD VIDÉO${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Test 1: Connexion admin
echo -e "${BLUE}[1/5] Connexion admin...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}')

if [ "$(echo "$LOGIN_RESPONSE" | jq -r .token 2>/dev/null)" != "null" ]; then
  ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r .token)
  echo -e "${GREEN}✓ Connexion admin réussie${NC}"
else
  echo -e "${RED}✗ Connexion admin échouée${NC}"
  echo -e "${YELLOW}  Response: $LOGIN_RESPONSE${NC}"
  exit 1
fi
echo ""

# Test 2: Vérifier les catégories disponibles
echo -e "${BLUE}[2/5] Vérification des catégories...${NC}"
CATEGORIES_RESPONSE=$(curl -s -X GET "$API_BASE_URL/categories" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
CATEGORIES_COUNT=$(echo "$CATEGORIES_RESPONSE" | jq '. | length' 2>/dev/null || echo "0")
echo -e "${GREEN}✓ Catégories disponibles: $CATEGORIES_COUNT${NC}"
echo ""

# Test 3: Créer un fichier vidéo de test (si pas de fichier fourni)
echo -e "${BLUE}[3/5] Préparation du fichier de test...${NC}"
TEST_VIDEO="/tmp/test_video.mp4"
if [ ! -f "$TEST_VIDEO" ]; then
  echo -e "${YELLOW}⚠ Fichier de test non trouvé. Création d'un fichier vide pour test...${NC}"
  # Créer un fichier minimal (juste pour tester l'endpoint)
  echo "test" > "$TEST_VIDEO"
fi
echo -e "${GREEN}✓ Fichier de test prêt: $TEST_VIDEO${NC}"
echo ""

# Test 4: Test d'upload (sans fichier réel, juste pour tester l'endpoint)
echo -e "${BLUE}[4/5] Test de l'endpoint upload...${NC}"
UPLOAD_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/admin/videos/upload" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "title=Test Video $(date +%s)" \
  -F "description=Description for test video" \
  -F "is_public=true" \
  -F "video=@$TEST_VIDEO;type=video/mp4")

UPLOAD_HTTP_CODE=$(echo "$UPLOAD_RESPONSE" | tail -n 1)
UPLOAD_BODY=$(echo "$UPLOAD_RESPONSE" | head -n -1)

if [ "$UPLOAD_HTTP_CODE" = "201" ]; then
  echo -e "${GREEN}✓ Upload de vidéo réussi (HTTP 201)${NC}"
  VIDEO_ID=$(echo "$UPLOAD_BODY" | jq -r '.video_id' 2>/dev/null)
  if [ "$VIDEO_ID" != "null" ] && [ -n "$VIDEO_ID" ]; then
    echo -e "${GREEN}  → Vidéo ID: $VIDEO_ID${NC}"
  fi
elif [ "$UPLOAD_HTTP_CODE" = "400" ]; then
  echo -e "${YELLOW}⚠ Upload échoué (HTTP 400) - Probablement le fichier de test invalide${NC}"
  echo -e "${YELLOW}  Response: $UPLOAD_BODY${NC}"
  echo -e "${YELLOW}  → C'est normal si le fichier n'est pas une vraie vidéo${NC}"
else
  echo -e "${RED}✗ Upload de vidéo échoué (HTTP $UPLOAD_HTTP_CODE)${NC}"
  echo -e "${YELLOW}  Response: $UPLOAD_BODY${NC}"
fi
echo ""

# Test 5: Vérifier la liste des vidéos (avec status=all pour voir toutes les vidéos y compris en traitement)
echo -e "${BLUE}[5/5] Vérification de la liste des vidéos...${NC}"
VIDEOS_RESPONSE=$(curl -s -X GET "$API_BASE_URL/videos?page=1&limit=100&status=all" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
VIDEOS_COUNT=$(echo "$VIDEOS_RESPONSE" | jq '.videos | length' 2>/dev/null || echo "0")
TOTAL_VIDEOS=$(echo "$VIDEOS_RESPONSE" | jq '.total' 2>/dev/null || echo "0")
echo -e "${GREEN}✓ Vidéos disponibles: $VIDEOS_COUNT (Total: $TOTAL_VIDEOS)${NC}"

# Afficher les détails des vidéos trouvées
if [ "$VIDEOS_COUNT" -gt 0 ]; then
  echo -e "${BLUE}  Détails des vidéos:${NC}"
  echo "$VIDEOS_RESPONSE" | jq -r '.videos[] | "    - \(.title) (ID: \(.id), Status: \(.status), Checksum: \(if .checksum then .checksum[0:16]+"..." else "N/A" end))"' 2>/dev/null || echo "    (Impossible d'afficher les détails)"
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  RÉSUMÉ${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✓ Tests d'upload terminés${NC}"
echo ""
echo "Pour tester avec un vrai fichier vidéo:"
echo "  1. Connectez-vous en tant qu'admin sur http://localhost:3300/login"
echo "  2. Allez sur http://localhost:3300/upload"
echo "  3. Sélectionnez un fichier vidéo et uploadez-le"
echo ""

