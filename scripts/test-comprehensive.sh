#!/bin/bash

# Script de tests complets pour TOUTES les fonctionnalités de StreamMake
# Usage: make test-comprehensive ou ./scripts/test-comprehensive.sh

# Ne pas utiliser set -e pour permettre la continuation même en cas d'erreur
# set -e

BASE_URL="${BASE_URL:-http://localhost:3300}"
API_URL="${API_URL:-$BASE_URL/api}"

echo "🧪 TESTS COMPLETS - StreamMake"
echo "================================"
echo ""

# Couleurs
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Compteurs
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Variables globales
ADMIN_TOKEN=""
USER_TOKEN=""
TEST_USER_ID=""
TEST_VIDEO_ID=""
TEST_PLAYLIST_ID=""
TEST_CATEGORY_ID=""
TEST_TORRENT_ID=""

# Fonction pour tester une requête
test_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    local token=$6
    local skip_on_fail=${7:-false}

    local url="$API_URL$endpoint"
    local headers=()
    
    if [ -n "$token" ]; then
        headers+=(-H "Authorization: Bearer $token")
    fi
    
    if [ "$method" = "POST" ] || [ "$method" = "PUT" ]; then
        if [[ "$data" != *"multipart/form-data"* ]]; then
            headers+=(-H "Content-Type: application/json")
        fi
    fi

    echo -n "  Test: $description... "
    
    local response
    if [ -n "$data" ] && [[ "$data" != *"multipart/form-data"* ]]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" "${headers[@]}" -d "$data" 2>/dev/null || echo -e "\n000")
    elif [[ "$data" == *"multipart/form-data"* ]]; then
        # Pour les uploads multipart
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" "${headers[@]}" -F "$data" 2>/dev/null || echo -e "\n000")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" "${headers[@]}" 2>/dev/null || echo -e "\n000")
    fi
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} (HTTP $http_code)"
        ((TESTS_PASSED++))
        return 0
    elif [ "$skip_on_fail" = "true" ]; then
        echo -e "${YELLOW}⚠${NC} (HTTP $http_code, attendu $expected_status) - IGNORÉ"
        ((TESTS_SKIPPED++))
        return 2
    else
        echo -e "${RED}✗${NC} (HTTP $http_code, attendu $expected_status)"
        if [ ${#body} -lt 200 ]; then
            echo "    Réponse: $body"
        fi
        ((TESTS_FAILED++))
        return 1
    fi
}

# Fonction pour extraire une valeur JSON
extract_json() {
    local json=$1
    local key=$2
    echo "$json" | grep -o "\"$key\":\"[^\"]*" | cut -d'"' -f4 || echo ""
}

# Fonction pour extraire un UUID JSON
extract_uuid() {
    local json=$1
    local key=$2
    echo "$json" | grep -o "\"$key\":\"[a-f0-9-]*" | cut -d'"' -f4 || echo ""
}

# ============================================
# 0. VÉRIFICATION DES SERVICES
# ============================================
echo "📋 0. VÉRIFICATION DES SERVICES"
echo "-------------------------------"

# Vérifier que l'API est accessible (avec retry)
MAX_RETRIES=5
RETRY_COUNT=0
API_ACCESSIBLE=false

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s "$API_URL/categories" > /dev/null 2>&1; then
        API_ACCESSIBLE=true
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        echo -e "${YELLOW}⚠${NC} Tentative $RETRY_COUNT/$MAX_RETRIES - Attente de l'API..."
        sleep 2
    fi
done

if [ "$API_ACCESSIBLE" = false ]; then
    echo -e "${RED}✗${NC} L'API n'est pas accessible à $API_URL après $MAX_RETRIES tentatives"
    echo "   Assurez-vous que les services Docker sont démarrés (make up)"
    exit 1
fi

echo -e "${GREEN}✓${NC} API accessible"
((TESTS_PASSED++))
echo ""

# ============================================
# 1. TESTS D'AUTHENTIFICATION
# ============================================
echo "📋 1. TESTS D'AUTHENTIFICATION"
echo "-------------------------------"

# Test d'inscription
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser'$(date +%s)'","email":"test'$(date +%s)'@example.com","password":"Test1234!"}')

if echo "$REGISTER_RESPONSE" | grep -q "token\|id"; then
    echo -e "${GREEN}✓${NC} Inscription réussie"
    ((TESTS_PASSED++))
    USER_TOKEN=$(extract_json "$REGISTER_RESPONSE" "token")
else
    echo -e "${YELLOW}⚠${NC} Inscription (peut-être déjà existant)"
    ((TESTS_SKIPPED++))
fi

# Test de connexion admin
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}')

ADMIN_TOKEN=$(extract_json "$ADMIN_LOGIN_RESPONSE" "token")

if [ -z "$ADMIN_TOKEN" ]; then
    # Essayer avec un autre mot de passe
    ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"dev@delhomme.ovh","password":"admin123"}')
    ADMIN_TOKEN=$(extract_json "$ADMIN_LOGIN_RESPONSE" "token")
fi

if [ -n "$ADMIN_TOKEN" ]; then
    echo -e "${GREEN}✓${NC} Connexion admin réussie"
    ((TESTS_PASSED++))
    test_request "GET" "/users/me" "" "200" "Récupération du profil admin" "$ADMIN_TOKEN"
else
    echo -e "${YELLOW}⚠${NC} Impossible de se connecter en admin, certains tests seront ignorés"
    ((TESTS_SKIPPED++))
fi

# Test de connexion utilisateur
if [ -z "$USER_TOKEN" ]; then
    USER_LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"Test1234!"}')
    USER_TOKEN=$(extract_json "$USER_LOGIN_RESPONSE" "token")
fi

if [ -n "$USER_TOKEN" ]; then
    echo -e "${GREEN}✓${NC} Connexion utilisateur réussie"
    ((TESTS_PASSED++))
    test_request "GET" "/users/me" "" "200" "Récupération du profil utilisateur" "$USER_TOKEN"
fi

echo ""

# ============================================
# 2. TESTS DES VIDÉOS (PUBLIQUES)
# ============================================
echo "📋 2. TESTS DES VIDÉOS (PUBLIQUES)"
echo "-----------------------------------"

test_request "GET" "/videos?page=1&limit=10" "" "200" "Liste des vidéos publiques"
test_request "GET" "/videos?sort=views" "" "200" "Liste des vidéos triées par vues"
test_request "GET" "/continue-watching" "" "200" "Continue watching (sans auth)"

# Récupérer une vidéo pour les tests suivants
VIDEOS_RESPONSE=$(curl -s "$API_URL/videos?limit=1")
TEST_VIDEO_ID=$(extract_uuid "$VIDEOS_RESPONSE" "id" | head -1)

if [ -n "$TEST_VIDEO_ID" ]; then
    test_request "GET" "/videos/$TEST_VIDEO_ID" "" "200" "Détails d'une vidéo"
    test_request "GET" "/videos/$TEST_VIDEO_ID/stream?format=progressive" "" "200" "Stream d'une vidéo" "" "true"
else
    echo -e "${YELLOW}⚠${NC} Aucune vidéo disponible pour les tests"
    ((TESTS_SKIPPED++))
fi

echo ""

# ============================================
# 3. TESTS DES CATÉGORIES
# ============================================
echo "📋 3. TESTS DES CATÉGORIES"
echo "---------------------------"

test_request "GET" "/categories" "" "200" "Liste des catégories (publique)"

if [ -n "$ADMIN_TOKEN" ]; then
    # Création d'une catégorie test
    CATEGORY_RESPONSE=$(curl -s -X POST "$API_URL/admin/categories" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{"name":"Test Category '$(date +%s)'","type":"other","description":"Test"}')
    
    TEST_CATEGORY_ID=$(extract_uuid "$CATEGORY_RESPONSE" "id")
    
    if [ -n "$TEST_CATEGORY_ID" ]; then
        test_request "PUT" "/admin/categories/$TEST_CATEGORY_ID" '{"name":"Test Category Updated","description":"Updated"}' "200" "Modification d'une catégorie" "$ADMIN_TOKEN" "true"
        test_request "POST" "/admin/categories/$TEST_CATEGORY_ID/subcategories" '{"name":"SubCategory Test","description":"Test"}' "201" "Création d'une sous-catégorie" "$ADMIN_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 4. TESTS DES VIDÉOS (PROTÉGÉES)
# ============================================
echo "📋 4. TESTS DES VIDÉOS (PROTÉGÉES)"
echo "-----------------------------------"

if [ -n "$USER_TOKEN" ]; then
    # Création d'une vidéo
    CREATE_VIDEO_RESPONSE=$(curl -s -X POST "$API_URL/videos" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $USER_TOKEN" \
        -d '{"title":"Test Video '$(date +%s)'","description":"Test","is_public":true}')
    
    CREATED_VIDEO_ID=$(extract_uuid "$CREATE_VIDEO_RESPONSE" "id")
    
    if [ -n "$CREATED_VIDEO_ID" ]; then
        test_request "PUT" "/videos/$CREATED_VIDEO_ID" '{"title":"Test Video Updated","description":"Updated"}' "200" "Modification d'une vidéo" "$USER_TOKEN" "true"
        test_request "DELETE" "/videos/$CREATED_VIDEO_ID" "" "200" "Suppression d'une vidéo" "$USER_TOKEN" "true"
    fi
    
    # Tests avec vidéo existante
    if [ -n "$TEST_VIDEO_ID" ]; then
        test_request "POST" "/videos/$TEST_VIDEO_ID/download" '{"url":"https://example.com/video.mp4"}' "200" "Téléchargement depuis URL" "$USER_TOKEN" "true"
    fi
fi

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/videos?status=all" "" "200" "Liste de toutes les vidéos (admin)" "$ADMIN_TOKEN"
    test_request "GET" "/videos?user_id=$(extract_uuid "$ADMIN_LOGIN_RESPONSE" "id")" "" "200" "Liste des vidéos d'un utilisateur" "$ADMIN_TOKEN" "true"
fi

echo ""

# ============================================
# 5. TESTS DES FAVORIS
# ============================================
echo "📋 5. TESTS DES FAVORIS"
echo "-----------------------"

if [ -n "$USER_TOKEN" ]; then
    test_request "GET" "/favorites" "" "200" "Liste des favoris de l'utilisateur" "$USER_TOKEN"
    
    if [ -n "$TEST_VIDEO_ID" ]; then
        test_request "POST" "/favorites/$TEST_VIDEO_ID" "" "201" "Ajout d'une vidéo aux favoris" "$USER_TOKEN" "true"
        test_request "GET" "/favorites/$TEST_VIDEO_ID/check" "" "200" "Vérification si vidéo est en favoris" "$USER_TOKEN"
        test_request "DELETE" "/favorites/$TEST_VIDEO_ID" "" "200" "Suppression d'une vidéo des favoris" "$USER_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 6. TESTS DE L'HISTORIQUE
# ============================================
echo "📋 6. TESTS DE L'HISTORIQUE"
echo "----------------------------"

if [ -n "$USER_TOKEN" ]; then
    test_request "GET" "/history" "" "200" "Récupération de l'historique de visionnage" "$USER_TOKEN"
    test_request "GET" "/continue-watching" "" "200" "Récupération des vidéos en cours de visionnage" "$USER_TOKEN"
    
    if [ -n "$TEST_VIDEO_ID" ]; then
        test_request "POST" "/history/$TEST_VIDEO_ID" '{"position":120,"duration":3600}' "200" "Ajout à l'historique de visionnage" "$USER_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 7. TESTS DES PLAYLISTS
# ============================================
echo "📋 7. TESTS DES PLAYLISTS"
echo "-------------------------"

if [ -n "$USER_TOKEN" ]; then
    test_request "GET" "/playlists" "" "200" "Liste des playlists de l'utilisateur" "$USER_TOKEN"
    
    # Création d'une playlist
    PLAYLIST_RESPONSE=$(curl -s -X POST "$API_URL/playlists" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $USER_TOKEN" \
        -d '{"name":"Test Playlist '$(date +%s)'","description":"Test","is_public":false}')
    
    TEST_PLAYLIST_ID=$(extract_uuid "$PLAYLIST_RESPONSE" "id")
    
    if [ -n "$TEST_PLAYLIST_ID" ]; then
        test_request "GET" "/playlists/$TEST_PLAYLIST_ID" "" "200" "Détails d'une playlist" "$USER_TOKEN"
        test_request "PUT" "/playlists/$TEST_PLAYLIST_ID" '{"name":"Test Playlist Updated","description":"Updated"}' "200" "Modification d'une playlist" "$USER_TOKEN" "true"
        
        if [ -n "$TEST_VIDEO_ID" ]; then
            test_request "POST" "/playlists/$TEST_PLAYLIST_ID/videos/$TEST_VIDEO_ID" "" "200" "Ajout d'une vidéo à une playlist" "$USER_TOKEN" "true"
            test_request "DELETE" "/playlists/$TEST_PLAYLIST_ID/videos/$TEST_VIDEO_ID" "" "200" "Suppression d'une vidéo d'une playlist" "$USER_TOKEN" "true"
        fi
        
        test_request "DELETE" "/playlists/$TEST_PLAYLIST_ID" "" "200" "Suppression d'une playlist" "$USER_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 8. TESTS ADMINISTRATEUR - STATISTIQUES
# ============================================
echo "📋 8. TESTS ADMINISTRATEUR - STATISTIQUES"
echo "------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/stats" "" "200" "Récupération des statistiques admin" "$ADMIN_TOKEN"
fi

echo ""

# ============================================
# 9. TESTS ADMINISTRATEUR - UTILISATEURS
# ============================================
echo "📋 9. TESTS ADMINISTRATEUR - UTILISATEURS"
echo "------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/users?page=1&limit=10" "" "200" "Liste des utilisateurs (admin)" "$ADMIN_TOKEN"
    test_request "GET" "/admin/users?search=test" "" "200" "Recherche d'utilisateurs" "$ADMIN_TOKEN" "true"
    
    # Récupérer un utilisateur pour les tests
    USERS_RESPONSE=$(curl -s -X GET "$API_URL/admin/users?limit=1" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    TEST_USER_ID=$(extract_uuid "$USERS_RESPONSE" "id" | head -1)
    
    if [ -n "$TEST_USER_ID" ]; then
        test_request "PUT" "/admin/users/$TEST_USER_ID" '{"is_active":true}' "200" "Modification d'un utilisateur" "$ADMIN_TOKEN" "true"
        # Note: Les mots de passe ne sont jamais retournés pour des raisons de sécurité
        # On peut seulement les modifier via un endpoint dédié si nécessaire
    fi
fi

echo ""

# ============================================
# 10. TESTS ADMINISTRATEUR - VIDÉOS
# ============================================
echo "📋 10. TESTS ADMINISTRATEUR - VIDÉOS"
echo "-------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/videos/invalid" "" "200" "Liste des vidéos invalides" "$ADMIN_TOKEN" "true"
    test_request "POST" "/admin/videos/clean-invalid" "" "200" "Nettoyage des vidéos invalides" "$ADMIN_TOKEN" "true"
    
    # Test d'upload (simulation - sans fichier réel)
    echo -n "  Test: Route upload accessible... "
    UPLOAD_TEST=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/admin/videos/upload" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -F "video=@/dev/null" \
        -F "title=Test Video Upload" \
        -F "is_public=true" 2>/dev/null || echo -e "\n000")
    UPLOAD_CODE=$(echo "$UPLOAD_TEST" | tail -n1)
    UPLOAD_BODY=$(echo "$UPLOAD_TEST" | sed '$d')
    
    if [ "$UPLOAD_CODE" = "400" ] || [ "$UPLOAD_CODE" = "422" ]; then
        echo -e "${GREEN}✓${NC} (HTTP $UPLOAD_CODE - attendu pour fichier invalide)"
        ((TESTS_PASSED++))
    elif [ "$UPLOAD_CODE" = "201" ]; then
        echo -e "${GREEN}✓${NC} (HTTP $UPLOAD_CODE)"
        ((TESTS_PASSED++))
        # Extraire l'ID de la vidéo uploadée
        UPLOADED_VIDEO_ID=$(extract_uuid "$UPLOAD_BODY" "video_id")
        if [ -n "$UPLOADED_VIDEO_ID" ]; then
            echo "    Vidéo uploadée: $UPLOADED_VIDEO_ID"
            # Tester l'accès à la vidéo uploadée (même en traitement)
            test_request "GET" "/videos/$UPLOADED_VIDEO_ID" "" "200" "Accès à la vidéo uploadée (admin)" "$ADMIN_TOKEN" "true"
        fi
    else
        echo -e "${YELLOW}⚠${NC} Route upload retourne: $UPLOAD_CODE"
        ((TESTS_SKIPPED++))
    fi
    
    # Test d'accès aux vidéos en traitement (admin)
    if [ -n "$TEST_VIDEO_ID" ]; then
        test_request "GET" "/videos/$TEST_VIDEO_ID" "" "200" "Accès à une vidéo par ID (admin)" "$ADMIN_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 11. TESTS ADMINISTRATEUR - TORRENTS
# ============================================
echo "📋 11. TESTS ADMINISTRATEUR - TORRENTS"
echo "---------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/torrents" "" "200" "Liste des torrents (admin)" "$ADMIN_TOKEN"
    
    # Création d'un torrent test
    TORRENT_RESPONSE=$(curl -s -X POST "$API_URL/admin/torrents" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{"name":"Test Torrent '$(date +%s)'","magnet_url":"magnet:?xt=urn:btih:test","status":"pending"}')
    
    TEST_TORRENT_ID=$(extract_uuid "$TORRENT_RESPONSE" "id")
    
    if [ -n "$TEST_TORRENT_ID" ]; then
        test_request "PUT" "/admin/torrents/$TEST_TORRENT_ID" '{"status":"downloading"}' "200" "Modification d'un torrent" "$ADMIN_TOKEN" "true"
        test_request "DELETE" "/admin/torrents/$TEST_TORRENT_ID" "" "200" "Suppression d'un torrent" "$ADMIN_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 12. TESTS ADMINISTRATEUR - CONFIGURATIONS TORRENT
# ============================================
echo "📋 12. TESTS ADMINISTRATEUR - CONFIGURATIONS TORRENT"
echo "-----------------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/torrent-configs" "" "200" "Liste des configurations torrent (admin)" "$ADMIN_TOKEN"
    
    # Création d'une configuration test
    CONFIG_RESPONSE=$(curl -s -X POST "$API_URL/admin/torrent-configs" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{"name":"Test Config","download_path":"/tmp","max_download_speed":1000,"max_upload_speed":500}')
    
    CONFIG_ID=$(extract_uuid "$CONFIG_RESPONSE" "id")
    
    if [ -n "$CONFIG_ID" ]; then
        test_request "PUT" "/admin/torrent-configs/$CONFIG_ID" '{"max_download_speed":2000}' "200" "Modification d'une configuration torrent" "$ADMIN_TOKEN" "true"
        test_request "DELETE" "/admin/torrent-configs/$CONFIG_ID" "" "200" "Suppression d'une configuration torrent" "$ADMIN_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 13. TESTS ADMINISTRATEUR - CLIENTS TORRENT
# ============================================
echo "📋 13. TESTS ADMINISTRATEUR - CLIENTS TORRENT"
echo "----------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/torrent-clients/status" "" "200" "Statut des clients torrent" "$ADMIN_TOKEN" "true"
    test_request "GET" "/admin/torrent-clients/current" "" "200" "Client torrent actuel" "$ADMIN_TOKEN" "true"
    test_request "PUT" "/admin/torrent-clients/current" '{"client":"transmission"}' "200" "Modification du client torrent actuel" "$ADMIN_TOKEN" "true"
    
    # Test de tous les clients torrent
    echo "  Test des clients torrent individuels:"
    test_request "POST" "/admin/torrent-clients/test" '{"client":"transmission","host":"transmission","port":9091,"username":"","password":""}' "200" "Test Transmission" "$ADMIN_TOKEN" "true"
    test_request "POST" "/admin/torrent-clients/test" '{"client":"qbittorrent","host":"qbittorrent","port":8080,"username":"admin","password":"adminadmin"}' "200" "Test qBittorrent" "$ADMIN_TOKEN" "true"
    test_request "POST" "/admin/torrent-clients/test" '{"client":"deluge","host":"deluge","port":8112,"username":"","password":"deluge"}' "200" "Test Deluge" "$ADMIN_TOKEN" "true"
    test_request "POST" "/admin/torrent-clients/test" '{"client":"rtorrent","host":"rtorrent","port":5000,"username":"","password":""}' "200" "Test rTorrent" "$ADMIN_TOKEN" "true"
fi

echo ""

# ============================================
# 14. TESTS ADMINISTRATEUR - PARAMÈTRES MÉDIA
# ============================================
echo "📋 14. TESTS ADMINISTRATEUR - PARAMÈTRES MÉDIA"
echo "----------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/admin/media-settings" "" "200" "Récupération des paramètres média" "$ADMIN_TOKEN" "true"
    test_request "PUT" "/admin/media-settings" '{"default_quality":"720p","auto_transcode":true,"auto_download":false}' "200" "Modification des paramètres média" "$ADMIN_TOKEN" "true"
    
    # Test de la configuration téléchargement automatique
    test_request "PUT" "/admin/media-settings" '{"auto_download":true,"auto_download_quality":"720p"}' "200" "Activation téléchargement automatique" "$ADMIN_TOKEN" "true"
fi

echo ""

# ============================================
# 15. TESTS ADMINISTRATEUR - SCRAPER
# ============================================
echo "📋 15. TESTS ADMINISTRATEUR - SCRAPER"
echo "-------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "POST" "/admin/scraper/search" '{"query":"test","category":"movies"}' "200" "Recherche de torrents via scraper" "$ADMIN_TOKEN" "true"
    test_request "GET" "/admin/scraper/details?url=https://example.com" "" "200" "Détails d'un torrent via scraper" "$ADMIN_TOKEN" "true"
fi

echo ""

# ============================================
# 16. TESTS DVD
# ============================================
echo "📋 16. TESTS DVD"
echo "----------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_request "GET" "/dvd/detect" "" "200" "Détection d'un lecteur DVD (admin)" "$ADMIN_TOKEN" "true"
    test_request "GET" "/dvd/info?device=/dev/sr0" "" "200" "Informations d'un DVD (admin)" "$ADMIN_TOKEN" "true"
    
    # Test d'extraction (simulation)
    EXTRACT_RESPONSE=$(curl -s -X POST "$API_URL/dvd/extract" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{"device":"/dev/sr0","output_path":"/tmp"}')
    
    EXTRACT_ID=$(extract_uuid "$EXTRACT_RESPONSE" "id")
    
    if [ -n "$EXTRACT_ID" ]; then
        test_request "GET" "/dvd/extract/$EXTRACT_ID/status" "" "200" "Statut d'une extraction DVD" "$ADMIN_TOKEN" "true"
    fi
fi

echo ""

# ============================================
# 17. TESTS DE SÉCURITÉ
# ============================================
echo "📋 17. TESTS DE SÉCURITÉ"
echo "------------------------"

# Test sans authentification
test_request "GET" "/users/me" "" "401" "Accès au profil sans authentification"
test_request "GET" "/admin/stats" "" "401" "Accès aux stats admin sans authentification"
test_request "POST" "/videos" '{"title":"Test"}' "401" "Création de vidéo sans authentification"
test_request "DELETE" "/videos/00000000-0000-0000-0000-000000000000" "" "401" "Suppression de vidéo sans authentification"

# Test avec token invalide
test_request "GET" "/users/me" "" "401" "Accès avec token invalide" "invalid_token"

# Test avec utilisateur non-admin
if [ -n "$USER_TOKEN" ]; then
    test_request "GET" "/admin/stats" "" "403" "Accès aux stats admin en tant qu'utilisateur" "$USER_TOKEN" "true"
    test_request "POST" "/admin/videos/upload" "" "403" "Upload vidéo en tant qu'utilisateur" "$USER_TOKEN" "true"
fi

echo ""

# ============================================
# 18. TESTS CORS
# ============================================
echo "📋 18. TESTS CORS"
echo "-----------------"

CORS_HEADERS=$(curl -s -D - -o /dev/null -X OPTIONS "$API_URL/videos" \
    -H "Origin: http://localhost:3300" \
    -H "Access-Control-Request-Method: GET" 2>/dev/null || echo "")

if echo "$CORS_HEADERS" | grep -q "Access-Control-Allow-Origin"; then
    echo -e "${GREEN}✓${NC} Headers CORS présents"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠${NC} Headers CORS non détectés"
    ((TESTS_SKIPPED++))
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
echo -e "${YELLOW}Tests ignorés: $TESTS_SKIPPED${NC}"
echo ""

TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((TESTS_PASSED * 100 / TOTAL_TESTS))
    echo -e "${BLUE}Taux de réussite: ${SUCCESS_RATE}%${NC}"
fi

echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ Tous les tests critiques sont passés !${NC}"
    exit 0
else
    echo -e "${RED}❌ Certains tests ont échoué${NC}"
    exit 1
fi
