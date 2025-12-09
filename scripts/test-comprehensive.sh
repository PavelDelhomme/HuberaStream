#!/bin/bash

# Script de tests complets pour toutes les fonctionnalités du backend et frontend
# Usage: make test-comprehensive ou ./scripts/test-comprehensive.sh

set -e

BASE_URL="${BASE_URL:-http://localhost:3300}"
API_URL="${API_URL:-$BASE_URL/api}"

echo "🧪 TESTS COMPLETS - StreamMake"
echo "================================"
echo ""

# Couleurs
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Compteurs
TESTS_PASSED=0
TESTS_FAILED=0

# Fonction pour tester une requête
test_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    local token=$6

    local url="$API_URL$endpoint"
    local headers=()
    
    if [ -n "$token" ]; then
        headers+=(-H "Authorization: Bearer $token")
    fi
    
    if [ "$method" = "POST" ] || [ "$method" = "PUT" ]; then
        headers+=(-H "Content-Type: application/json")
    fi

    echo -n "  Test: $description... "
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" "${headers[@]}" -d "$data" 2>/dev/null || echo -e "\n000")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" "${headers[@]}" 2>/dev/null || echo -e "\n000")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} (HTTP $http_code)"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} (HTTP $http_code, attendu $expected_status)"
        echo "    Réponse: $body"
        ((TESTS_FAILED++))
        return 1
    fi
}

# ============================================
# 1. TESTS D'AUTHENTIFICATION
# ============================================
echo "📋 1. TESTS D'AUTHENTIFICATION"
echo "-------------------------------"

# Test d'inscription
test_request "POST" "/auth/register" '{"username":"testuser","email":"test@example.com","password":"Test1234!"}' "201" "Inscription d'un nouvel utilisateur"

# Test de connexion
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"Test1234!"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    # Essayer avec dev@delhomme.ovh
    LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"dev@delhomme.ovh","password":"admin123"}')
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
fi

if [ -n "$TOKEN" ]; then
    test_request "GET" "/users/me" "" "200" "Récupération du profil utilisateur" "$TOKEN"
else
    echo -e "${YELLOW}⚠${NC}  Impossible de récupérer un token, certains tests seront ignorés"
fi

echo ""

# ============================================
# 2. TESTS DES VIDÉOS
# ============================================
echo "📋 2. TESTS DES VIDÉOS"
echo "----------------------"

if [ -n "$TOKEN" ]; then
    test_request "GET" "/videos?page=1&limit=10" "" "200" "Liste des vidéos publiques"
    test_request "GET" "/videos?status=all" "" "200" "Liste de toutes les vidéos (admin)" "$TOKEN"
    test_request "GET" "/videos?sort=views" "" "200" "Liste des vidéos triées par vues"
else
    test_request "GET" "/videos?page=1&limit=10" "" "200" "Liste des vidéos publiques"
fi

echo ""

# ============================================
# 3. TESTS DES CATÉGORIES
# ============================================
echo "📋 3. TESTS DES CATÉGORIES"
echo "--------------------------"

test_request "GET" "/categories" "" "200" "Liste des catégories (publique)"

if [ -n "$TOKEN" ]; then
    # Test de création de catégorie (admin)
    test_request "POST" "/admin/categories" '{"name":"Test Category","type":"other","description":"Test"}' "201" "Création d'une catégorie (admin)" "$TOKEN" || true
fi

echo ""

# ============================================
# 4. TESTS DES FAVORIS
# ============================================
echo "📋 4. TESTS DES FAVORIS"
echo "-----------------------"

if [ -n "$TOKEN" ]; then
    test_request "GET" "/favorites" "" "200" "Liste des favoris de l'utilisateur" "$TOKEN"
    
    # Récupérer une vidéo pour tester l'ajout aux favoris
    VIDEOS_RESPONSE=$(curl -s -X GET "$API_URL/videos?limit=1" -H "Authorization: Bearer $TOKEN")
    VIDEO_ID=$(echo "$VIDEOS_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    
    if [ -n "$VIDEO_ID" ]; then
        test_request "POST" "/favorites/$VIDEO_ID" "" "201" "Ajout d'une vidéo aux favoris" "$TOKEN" || true
        test_request "GET" "/favorites/$VIDEO_ID/check" "" "200" "Vérification si vidéo est en favoris" "$TOKEN"
        test_request "DELETE" "/favorites/$VIDEO_ID" "" "200" "Suppression d'une vidéo des favoris" "$TOKEN" || true
    else
        echo -e "${YELLOW}⚠${NC}  Aucune vidéo disponible pour tester les favoris"
    fi
else
    echo -e "${YELLOW}⚠${NC}  Tests des favoris ignorés (pas de token)"
fi

echo ""

# ============================================
# 5. TESTS DE L'HISTORIQUE
# ============================================
echo "📋 5. TESTS DE L'HISTORIQUE"
echo "----------------------------"

if [ -n "$TOKEN" ]; then
    test_request "GET" "/history" "" "200" "Récupération de l'historique de visionnage" "$TOKEN"
    test_request "GET" "/continue-watching" "" "200" "Récupération des vidéos en cours de visionnage" "$TOKEN"
else
    echo -e "${YELLOW}⚠${NC}  Tests de l'historique ignorés (pas de token)"
fi

echo ""

# ============================================
# 6. TESTS DES PLAYLISTS
# ============================================
echo "📋 6. TESTS DES PLAYLISTS"
echo "-------------------------"

if [ -n "$TOKEN" ]; then
    test_request "GET" "/playlists" "" "200" "Liste des playlists de l'utilisateur" "$TOKEN"
    test_request "POST" "/playlists" '{"name":"Test Playlist","description":"Test","is_public":false}' "201" "Création d'une playlist" "$TOKEN" || true
else
    echo -e "${YELLOW}⚠${NC}  Tests des playlists ignorés (pas de token)"
fi

echo ""

# ============================================
# 7. TESTS ADMINISTRATEUR
# ============================================
echo "📋 7. TESTS ADMINISTRATEUR"
echo "--------------------------"

if [ -n "$TOKEN" ]; then
    test_request "GET" "/admin/stats" "" "200" "Récupération des statistiques admin" "$TOKEN" || true
    test_request "GET" "/admin/users?page=1&limit=10" "" "200" "Liste des utilisateurs (admin)" "$TOKEN" || true
    test_request "GET" "/admin/torrents" "" "200" "Liste des torrents (admin)" "$TOKEN" || true
    test_request "GET" "/admin/torrent-configs" "" "200" "Liste des configurations torrent (admin)" "$TOKEN" || true
else
    echo -e "${YELLOW}⚠${NC}  Tests admin ignorés (pas de token)"
fi

echo ""

# ============================================
# 8. TESTS DE L'EXTRACTION DVD
# ============================================
echo "📋 8. TESTS DE L'EXTRACTION DVD"
echo "-------------------------------"

if [ -n "$TOKEN" ]; then
    test_request "GET" "/dvd/detect" "" "200" "Détection d'un lecteur DVD (admin)" "$TOKEN" || true
else
    echo -e "${YELLOW}⚠${NC}  Tests DVD ignorés (pas de token)"
fi

echo ""

# ============================================
# RÉSUMÉ
# ============================================
echo "================================"
echo "📊 RÉSUMÉ DES TESTS"
echo "================================"
echo -e "${GREEN}Tests réussis: $TESTS_PASSED${NC}"
echo -e "${RED}Tests échoués: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ Tous les tests sont passés !${NC}"
    exit 0
else
    echo -e "${RED}❌ Certains tests ont échoué${NC}"
    exit 1
fi

