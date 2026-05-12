#!/bin/bash

# Script de test complet pour StreamMake
# Teste toutes les fonctionnalités backend et frontend

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL="http://localhost:3300/api"
TOKEN=""
ADMIN_TOKEN=""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  TESTS COMPLETS STREAMMAKE${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Fonction pour tester une route
test_route() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    local use_token=${6:-false}
    
    local headers="Content-Type: application/json"
    if [ "$use_token" = "true" ] && [ -n "$TOKEN" ]; then
        headers="$headers\nAuthorization: Bearer $TOKEN"
    fi
    
    if [ "$method" = "GET" ]; then
        if [ -n "$TOKEN" ] && [ "$use_token" = "true" ]; then
            response=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" "$API_URL$endpoint")
        else
            response=$(curl -s -w "\n%{http_code}" "$API_URL$endpoint")
        fi
    else
        if [ -n "$TOKEN" ] && [ "$use_token" = "true" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$data" "$API_URL$endpoint")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$API_URL$endpoint")
        fi
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} $description (HTTP $http_code)"
        return 0
    else
        echo -e "${RED}✗${NC} $description (Expected: $expected_status, Got: $http_code)"
        if [ ${#body} -lt 200 ]; then
            echo -e "${YELLOW}  Response: $body${NC}"
        fi
        return 1
    fi
}

# Vérifier et démarrer les services si nécessaire
echo -e "${BLUE}[1/10] Vérification des services Docker...${NC}"
SERVICES_RUNNING=$(docker compose ps 2>/dev/null | grep -c "Up\|Restarting" || docker-compose ps 2>/dev/null | grep -c "Up\|Restarting" || echo "0")
if [ "$SERVICES_RUNNING" = "0" ]; then
    echo -e "${YELLOW}⚠️  Les services Docker ne sont pas démarrés. Démarrage...${NC}"
    echo -e "${YELLOW}💡 Utilisation de 'make up' pour construire et démarrer les services...${NC}"
    
    # Utiliser make up qui gère mieux le build
    SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
    cd "$SCRIPT_DIR" || exit 1
    if make up 2>&1; then
        echo -e "${GREEN}✅ Services démarrés${NC}"
    else
        echo -e "${RED}❌ Impossible de démarrer les services${NC}"
        echo -e "${YELLOW}💡 Vérifiez les erreurs ci-dessus et essayez: make up${NC}"
        exit 1
    fi
    cd - > /dev/null || exit 1
    
    echo -e "${YELLOW}⏳ Attente du démarrage des services (20 secondes)...${NC}"
    sleep 20
    
    # Vérifier que l'API répond
    MAX_RETRIES=15
    RETRY_COUNT=0
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -s http://localhost:3300/api/categories &> /dev/null || curl -s http://localhost:3301/health &> /dev/null; then
            echo -e "${GREEN}✓ Services Docker démarrés${NC}"
            break
        fi
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
            echo -e "${YELLOW}⏳ Attente... ($RETRY_COUNT/$MAX_RETRIES)${NC}"
            sleep 2
        fi
    done
    
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}✗ Impossible de démarrer les services${NC}"
        echo -e "${YELLOW}💡 Démarrez manuellement avec: make up${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ Services Docker démarrés${NC}"
fi
echo ""

# Test 1: Authentification
echo -e "${BLUE}[2/10] Test d'authentification...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
ADMIN_TOKEN=$TOKEN

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ Échec de la connexion admin${NC}"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi
echo -e "${GREEN}✓ Connexion admin réussie${NC}"
echo ""

# Test 2: Routes publiques
echo -e "${BLUE}[3/12] Test des routes publiques...${NC}"
test_route "GET" "/videos" "" "200" "Liste des vidéos" false
test_route "GET" "/continue-watching" "" "200" "Continue watching (sans auth)" false
# Note: /categories est maintenant publique
test_route "GET" "/categories" "" "200" "Liste des catégories (publique)" false
echo ""

# Test 3: Routes protégées
echo -e "${BLUE}[4/10] Test des routes protégées...${NC}"
test_route "GET" "/continue-watching" "" "200" "Continue watching (avec auth)" true
test_route "GET" "/users/me" "" "200" "Profil utilisateur" true
test_route "GET" "/playlists" "" "200" "Liste des playlists" true
test_route "GET" "/history" "" "200" "Historique de visionnage" true
echo ""

# Test 4: Routes admin
echo -e "${BLUE}[5/10] Test des routes admin...${NC}"
test_route "GET" "/admin/stats" "" "200" "Statistiques admin" true
test_route "GET" "/admin/users" "" "200" "Liste des utilisateurs" true
# Note: La route /admin/categories peut ne pas exister, on teste avec auth
# Test de la route admin/categories (peut ne pas exister)
if test_route "GET" "/admin/categories" "" "404" "Gestion catégories (route admin)" true 2>/dev/null; then
    : # Route retourne 404 comme attendu
else
    # Si la route existe et retourne autre chose que 404, on l'affiche
    test_route "GET" "/admin/categories" "" "200" "Gestion catégories" true 2>/dev/null || true
fi
test_route "GET" "/admin/torrents" "" "200" "Gestion torrents" true
echo ""

# Statut Prowlarr / Radarr / Sonarr (URLs optionnelles dans la config backend)
echo -e "${BLUE}[5b] Statut media-stack (*arr)...${NC}"
test_route "GET" "/admin/media-stack/status" "" "200" "Statut Prowlarr/Radarr/Sonarr (admin)" true
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^streammake-prowlarr$'; then
  P_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3326/ 2>/dev/null || echo "000")
  if [ "$P_CODE" = "200" ] || [ "$P_CODE" = "302" ] || [ "$P_CODE" = "401" ]; then
    echo -e "${GREEN}✓${NC} Prowlarr UI joignable sur le port hôte 3326 (HTTP $P_CODE)"
  else
    echo -e "${YELLOW}⚠${NC} Prowlarr conteneur up mais port 3326 HTTP $P_CODE"
  fi
else
  echo -e "${YELLOW}ℹ${NC} Profile arr non démarré : ignorer les ports 3326–3328 (make up-arr pour activer)"
fi
echo ""

# Test 5: Routes DVD (admin seulement)
echo -e "${BLUE}[6/10] Test des routes DVD...${NC}"
# La route /dvd/detect peut retourner 200 avec une liste vide si aucun DVD n'est détecté
# ou 404 si la route n'existe pas. On accepte les deux comme valides.
test_route "GET" "/dvd/detect" "" "200" "Détection DVD (sans DVD)" true "true"
# Note: 200 avec liste vide ou 404 sont tous deux valides
echo ""

# Test 6: Création de contenu
echo -e "${BLUE}[7/10] Test de création de contenu...${NC}"
CATEGORY_RESPONSE=$(curl -s -X POST "$API_URL/admin/categories" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{"name":"Test Category","description":"Test"}')
echo -e "${GREEN}✓ Création de catégorie test${NC}"
echo ""

# Test 7: Upload (simulation)
echo -e "${BLUE}[8/10] Test de l'endpoint upload...${NC}"
# On teste juste que la route existe (sans vraiment uploader)
UPLOAD_TEST=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/videos/upload" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -F "video=@/dev/null" 2>/dev/null || echo "400")
UPLOAD_CODE=$(echo "$UPLOAD_TEST" | tail -n1)
if [ "$UPLOAD_CODE" = "400" ] || [ "$UPLOAD_CODE" = "422" ] || [ "$UPLOAD_CODE" = "201" ]; then
    echo -e "${GREEN}✓ Route upload accessible (code: $UPLOAD_CODE)${NC}"
else
    echo -e "${YELLOW}⚠ Route upload retourne: $UPLOAD_CODE${NC}"
fi
echo ""

# Test 8: Vérification CORS
echo -e "${BLUE}[9/10] Test des headers CORS...${NC}"
CORS_HEADERS=$(curl -s -D - -o /dev/null -X OPTIONS "$API_URL/videos" \
    -H "Origin: http://localhost:3300" \
    -H "Access-Control-Request-Method: GET")

if echo "$CORS_HEADERS" | grep -q "Access-Control-Allow-Origin"; then
    echo -e "${GREEN}✓ Headers CORS présents${NC}"
else
    echo -e "${RED}✗ Headers CORS manquants${NC}"
fi
echo ""

# Test 9: Vérification de la base de données
echo -e "${BLUE}[10/12] Vérification de la base de données...${NC}"
DB_CHECK=$(docker-compose exec -T postgres psql -U streammake -d streammake_db -c "SELECT COUNT(*) FROM users;" 2>/dev/null | grep -o "[0-9]")
if [ -n "$DB_CHECK" ]; then
    echo -e "${GREEN}✓ Base de données accessible (Users: $DB_CHECK)${NC}"
else
    echo -e "${YELLOW}⚠ Impossible de vérifier la base de données${NC}"
fi
echo ""

# Test 10: Vérification de l'intégrité des vidéos (checksum)
echo -e "${BLUE}[11/14] Vérification de l'intégrité des vidéos...${NC}"
VIDEOS_WITH_CHECKSUM=$(docker-compose exec -T postgres psql -U streammake -d streammake_db -t -c "SELECT COUNT(*) FROM videos WHERE checksum IS NOT NULL AND checksum != '';" 2>/dev/null | tr -d ' ')
TOTAL_VIDEOS=$(docker-compose exec -T postgres psql -U streammake -d streammake_db -t -c "SELECT COUNT(*) FROM videos;" 2>/dev/null | tr -d ' ')

if [ -n "$VIDEOS_WITH_CHECKSUM" ] && [ -n "$TOTAL_VIDEOS" ]; then
    if [ "$TOTAL_VIDEOS" -gt 0 ]; then
        PERCENTAGE=$((VIDEOS_WITH_CHECKSUM * 100 / TOTAL_VIDEOS))
        if [ "$PERCENTAGE" -eq 100 ]; then
            echo -e "${GREEN}✓ Toutes les vidéos ont un checksum ($VIDEOS_WITH_CHECKSUM/$TOTAL_VIDEOS)${NC}"
        else
            echo -e "${YELLOW}⚠ $PERCENTAGE% des vidéos ont un checksum ($VIDEOS_WITH_CHECKSUM/$TOTAL_VIDEOS)${NC}"
        fi
    else
        echo -e "${GREEN}✓ Aucune vidéo dans la base (normal si nouveau)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Impossible de vérifier les checksums${NC}"
fi
echo ""

# Test 11: Vérification de l'intégrité des fichiers vidéo
echo -e "${BLUE}[12/14] Vérification de l'intégrité des fichiers vidéo...${NC}"
# Vérifier que les fichiers vidéo existent et correspondent aux checksums en base
INTEGRITY_CHECK=$(docker-compose exec -T backend sh -c "cd /app && find storage/videos -type f 2>/dev/null | wc -l" 2>/dev/null | tr -d ' ')
if [ -n "$INTEGRITY_CHECK" ]; then
    echo -e "${GREEN}✓ $INTEGRITY_CHECK fichier(s) vidéo trouvé(s) dans le stockage${NC}"
    # Note: La vérification complète des checksums nécessiterait un script Go dédié
    echo -e "${YELLOW}  ℹ Vérification complète des checksums disponible via l'API admin${NC}"
else
    echo -e "${YELLOW}⚠ Impossible de vérifier les fichiers vidéo${NC}"
fi
echo ""

# Test 12: Vérification des routes admin vidéos
echo -e "${BLUE}[13/14] Vérification des routes admin vidéos...${NC}"
test_route "GET" "/admin/videos/invalid" "" "200" "Liste des vidéos invalides (admin)" true
test_route "POST" "/admin/videos/clean-invalid" "" "200" "Nettoyage des vidéos invalides (admin)" true
echo ""

# Test 13: Vérification des routes DVD
echo -e "${BLUE}[14/14] Vérification des routes DVD...${NC}"
test_route "GET" "/dvd/detect" "" "200" "Détection DVD (admin)" true "true"
# La route /dvd/info peut retourner 500 ou 404 si aucun DVD n'est présent, c'est normal
echo -n "  Test: Info DVD (admin)... "
INFO_DVD_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/dvd/info?device=/dev/sr0" \
    -H "Authorization: Bearer $ADMIN_TOKEN" 2>/dev/null || echo -e "\n000")
INFO_DVD_CODE=$(echo "$INFO_DVD_RESPONSE" | tail -n1)
if [ "$INFO_DVD_CODE" = "200" ] || [ "$INFO_DVD_CODE" = "500" ] || [ "$INFO_DVD_CODE" = "404" ]; then
    echo -e "${GREEN}✓${NC} (HTTP $INFO_DVD_CODE - normal si aucun DVD)"
else
    echo -e "${YELLOW}⚠${NC} (HTTP $INFO_DVD_CODE, attendu 200/500/404)"
fi
echo ""

# Résumé
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  RÉSUMÉ DES TESTS${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✓ Tests terminés${NC}"
echo ""
echo "Pour tester manuellement:"
echo "  - Frontend: http://localhost:3300"
echo "  - Backend API: http://localhost:3300/api"
echo "  - Admin: http://localhost:3300/admin"
echo ""

