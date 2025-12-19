#!/bin/bash

# Script de test pour les index de torrents

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL="http://localhost:3300/api"
ADMIN_EMAIL="dev@delhomme.ovh"
ADMIN_PASSWORD="AdminStreamMake2024!"

echo -e "${BLUE}==========================================${NC}"
echo -e "${BLUE}Test des Index de Torrents${NC}"
echo -e "${BLUE}==========================================${NC}"
echo ""

# 1. Connexion admin
echo -e "${YELLOW}[1/4] Connexion admin...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ Échec de la connexion${NC}"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi
echo -e "${GREEN}✓ Connexion réussie${NC}"
echo ""

# 2. Lister les index
echo -e "${YELLOW}[2/4] Liste des index...${NC}"
INDEXES_RESPONSE=$(curl -s -X GET "$API_URL/admin/torrent-indexes" \
    -H "Authorization: Bearer $TOKEN")

INDEX_COUNT=$(echo "$INDEXES_RESPONSE" | grep -o '"id":"[^"]*"' | wc -l)

if [ "$INDEX_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}⚠ Aucun index trouvé${NC}"
    echo "Response: $INDEXES_RESPONSE"
else
    echo -e "${GREEN}✓ $INDEX_COUNT index trouvé(s)${NC}"
    # Afficher les IDs des index
    echo "$INDEXES_RESPONSE" | grep -o '"id":"[^"]*"' | head -3 | while read -r line; do
        ID=$(echo "$line" | cut -d'"' -f4)
        echo "  - ID: $ID"
    done
fi
echo ""

# 3. Tester un index (si disponible)
if [ "$INDEX_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}[3/4] Test d'un index...${NC}"
    FIRST_INDEX_ID=$(echo "$INDEXES_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [ -n "$FIRST_INDEX_ID" ]; then
        TEST_RESPONSE=$(curl -s -X POST "$API_URL/admin/torrent-indexes/$FIRST_INDEX_ID/test" \
            -H "Authorization: Bearer $TOKEN")
        
        SUCCESS=$(echo "$TEST_RESPONSE" | grep -o '"success":[^,}]*' | cut -d':' -f2)
        RESULTS_COUNT=$(echo "$TEST_RESPONSE" | grep -o '"results_count":[^,}]*' | cut -d':' -f2)
        
        if [ "$SUCCESS" = "true" ]; then
            echo -e "${GREEN}✓ Test réussi${NC}"
            echo -e "${GREEN}  Résultats trouvés: $RESULTS_COUNT${NC}"
            
            # Vérifier si les résultats sont présents
            HAS_RESULTS=$(echo "$TEST_RESPONSE" | grep -c '"results":\[')
            if [ "$HAS_RESULTS" -gt 0 ]; then
                echo -e "${GREEN}✓ Résultats détaillés présents dans la réponse${NC}"
                # Compter les résultats
                RESULT_NAMES=$(echo "$TEST_RESPONSE" | grep -o '"Name":"[^"]*"' | wc -l)
                echo -e "${GREEN}  Noms de torrents trouvés: $RESULT_NAMES${NC}"
            else
                echo -e "${YELLOW}⚠ Résultats détaillés non présents${NC}"
            fi
        else
            echo -e "${RED}✗ Test échoué${NC}"
            echo "Response: $TEST_RESPONSE"
        fi
    else
        echo -e "${YELLOW}⚠ Impossible d'extraire l'ID de l'index${NC}"
    fi
else
    echo -e "${YELLOW}[3/4] Test d'index - SKIP (aucun index disponible)${NC}"
fi
echo ""

# 4. Vérifier la structure de la réponse
echo -e "${YELLOW}[4/4] Vérification de la structure de la réponse...${NC}"
if [ "$INDEX_COUNT" -gt 0 ] && [ -n "$FIRST_INDEX_ID" ]; then
    TEST_RESPONSE=$(curl -s -X POST "$API_URL/admin/torrent-indexes/$FIRST_INDEX_ID/test" \
        -H "Authorization: Bearer $TOKEN")
    
    # Vérifier les champs attendus
    HAS_SUCCESS=$(echo "$TEST_RESPONSE" | grep -c '"success"')
    HAS_MESSAGE=$(echo "$TEST_RESPONSE" | grep -c '"message"')
    HAS_RESULTS_COUNT=$(echo "$TEST_RESPONSE" | grep -c '"results_count"')
    HAS_RESULTS=$(echo "$TEST_RESPONSE" | grep -c '"results"')
    
    if [ "$HAS_SUCCESS" -gt 0 ] && [ "$HAS_MESSAGE" -gt 0 ] && [ "$HAS_RESULTS_COUNT" -gt 0 ] && [ "$HAS_RESULTS" -gt 0 ]; then
        echo -e "${GREEN}✓ Structure de réponse correcte${NC}"
        echo -e "${GREEN}  - success: ✓${NC}"
        echo -e "${GREEN}  - message: ✓${NC}"
        echo -e "${GREEN}  - results_count: ✓${NC}"
        echo -e "${GREEN}  - results: ✓${NC}"
    else
        echo -e "${RED}✗ Structure de réponse incomplète${NC}"
        echo "  - success: $([ "$HAS_SUCCESS" -gt 0 ] && echo "✓" || echo "✗")"
        echo "  - message: $([ "$HAS_MESSAGE" -gt 0 ] && echo "✓" || echo "✗")"
        echo "  - results_count: $([ "$HAS_RESULTS_COUNT" -gt 0 ] && echo "✓" || echo "✗")"
        echo "  - results: $([ "$HAS_RESULTS" -gt 0 ] && echo "✓" || echo "✗")"
    fi
else
    echo -e "${YELLOW}⚠ SKIP (aucun index disponible)${NC}"
fi
echo ""

echo -e "${BLUE}==========================================${NC}"
echo -e "${GREEN}Tests terminés${NC}"
echo -e "${BLUE}==========================================${NC}"

