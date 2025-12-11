#!/bin/bash

# Script de test pour la recherche de métadonnées
# Usage: make test-metadata

set -e

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🧪 Test de recherche de métadonnées pour RED_ONE.webm${NC}"
echo ""

# Vérifier que le fichier existe
VIDEO_FILE="$HOME/Vidéos/RED_ONE.webm"
if [ ! -f "$VIDEO_FILE" ]; then
    echo -e "${RED}❌ Fichier non trouvé: $VIDEO_FILE${NC}"
    echo -e "${YELLOW}💡 Créez le fichier ou modifiez le chemin dans le script${NC}"
    exit 1
fi

echo -e "${GREEN}📁 Fichier trouvé: $VIDEO_FILE${NC}"
echo ""

# Vérifier que le backend est démarré
if ! docker-compose ps | grep -q "streammake-backend.*Up"; then
    echo -e "${YELLOW}⚠️  Le backend n'est pas démarré. Démarrage...${NC}"
    docker-compose up -d backend
    sleep 5
fi

# Obtenir le token d'authentification (admin)
echo -e "${GREEN}🔐 Authentification...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:3300/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ Échec de l'authentification${NC}"
    echo "Réponse: $LOGIN_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✅ Authentifié${NC}"
echo ""

# Tester la recherche de métadonnées
echo -e "${GREEN}🔍 Recherche de métadonnées pour: RED_ONE.webm${NC}"
SEARCH_RESPONSE=$(curl -s -X POST http://localhost:3300/api/admin/videos/search-metadata \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"filename":"RED_ONE.webm"}')

echo -e "${YELLOW}Réponse:${NC}"
echo "$SEARCH_RESPONSE" | jq '.' 2>/dev/null || echo "$SEARCH_RESPONSE"
echo ""

# Vérifier les résultats
RESULT_COUNT=$(echo "$SEARCH_RESPONSE" | jq '.count // 0' 2>/dev/null || echo "0")

if [ "$RESULT_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ $RESULT_COUNT résultat(s) trouvé(s)${NC}"
    echo ""
    
    # Afficher tous les résultats de manière détaillée
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}📋 RÉSULTATS TROUVÉS SUR INTERNET (en temps réel)${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Parcourir tous les résultats
    for i in $(seq 0 $((RESULT_COUNT - 1))); do
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}🎬 RÉSULTAT #$((i + 1))${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        
        RESULT=$(echo "$SEARCH_RESPONSE" | jq ".results[$i]" 2>/dev/null)
        
        # Titre
        TITLE=$(echo "$RESULT" | jq -r '.title // "N/A"' 2>/dev/null)
        ORIGINAL_TITLE=$(echo "$RESULT" | jq -r '.original_title // "N/A"' 2>/dev/null)
        echo -e "${YELLOW}📽️  Titre:${NC} $TITLE"
        if [ "$ORIGINAL_TITLE" != "N/A" ] && [ "$ORIGINAL_TITLE" != "$TITLE" ]; then
            echo -e "${YELLOW}   Titre original:${NC} $ORIGINAL_TITLE"
        fi
        echo ""
        
        # Réalisateur
        DIRECTOR=$(echo "$RESULT" | jq -r '.director // ""' 2>/dev/null)
        if [ -n "$DIRECTOR" ] && [ "$DIRECTOR" != "" ]; then
            echo -e "${YELLOW}🎭 Réalisateur:${NC} $DIRECTOR"
        fi
        
        # Acteurs
        ACTORS=$(echo "$RESULT" | jq -r '.actors // []' 2>/dev/null)
        if [ "$ACTORS" != "[]" ] && [ "$ACTORS" != "null" ] && [ -n "$ACTORS" ]; then
            ACTORS_STR=$(echo "$ACTORS" | jq -r 'join(", ")' 2>/dev/null || echo "$ACTORS")
            if [ -n "$ACTORS_STR" ] && [ "$ACTORS_STR" != "" ]; then
                echo -e "${YELLOW}👥 Acteurs:${NC} $ACTORS_STR"
            fi
        fi
        echo ""
        
        # Année et durée
        YEAR=$(echo "$RESULT" | jq -r '.production_year // 0' 2>/dev/null)
        RUNTIME=$(echo "$RESULT" | jq -r '.runtime // 0' 2>/dev/null)
        RELEASE_DATE=$(echo "$RESULT" | jq -r '.release_date // ""' 2>/dev/null)
        
        if [ "$YEAR" != "0" ] && [ "$YEAR" != "null" ]; then
            echo -e "${YELLOW}📅 Année:${NC} $YEAR"
        fi
        if [ -n "$RELEASE_DATE" ] && [ "$RELEASE_DATE" != "" ] && [ "$RELEASE_DATE" != "null" ]; then
            echo -e "${YELLOW}📆 Date de sortie:${NC} $RELEASE_DATE"
        fi
        if [ "$RUNTIME" != "0" ] && [ "$RUNTIME" != "null" ] && [ "$RUNTIME" != "" ]; then
            HOURS=$((RUNTIME / 60))
            MINUTES=$((RUNTIME % 60))
            echo -e "${YELLOW}⏱️  Durée:${NC} ${HOURS}h ${MINUTES}min"
        fi
        echo ""
        
        # Genres
        GENRES=$(echo "$RESULT" | jq -r '.genre // []' 2>/dev/null)
        if [ "$GENRES" != "[]" ] && [ "$GENRES" != "null" ] && [ -n "$GENRES" ]; then
            GENRES_STR=$(echo "$GENRES" | jq -r 'join(", ")' 2>/dev/null || echo "$GENRES")
            if [ -n "$GENRES_STR" ] && [ "$GENRES_STR" != "" ]; then
                echo -e "${YELLOW}🎨 Genres:${NC} $GENRES_STR"
                echo ""
            fi
        fi
        
        # Note
        RATING=$(echo "$RESULT" | jq -r '.rating // 0' 2>/dev/null)
        if [ "$RATING" != "0" ] && [ "$RATING" != "null" ] && [ "$RATING" != "" ]; then
            echo -e "${YELLOW}⭐ Note:${NC} $RATING/10"
            echo ""
        fi
        
        # Synopsis
        SYNOPSIS=$(echo "$RESULT" | jq -r '.synopsis // ""' 2>/dev/null)
        DESCRIPTION=$(echo "$RESULT" | jq -r '.description // ""' 2>/dev/null)
        if [ -n "$SYNOPSIS" ] && [ "$SYNOPSIS" != "" ] && [ "$SYNOPSIS" != "null" ] && [ "$SYNOPSIS" != "Vidéo uploadée: "* ]; then
            echo -e "${YELLOW}📖 Synopsis:${NC}"
            echo "$SYNOPSIS" | fold -w 80 -s | sed 's/^/   /'
            echo ""
        elif [ -n "$DESCRIPTION" ] && [ "$DESCRIPTION" != "" ] && [ "$DESCRIPTION" != "null" ] && [ "$DESCRIPTION" != "Vidéo: "* ]; then
            echo -e "${YELLOW}📖 Description:${NC}"
            echo "$DESCRIPTION" | fold -w 80 -s | sed 's/^/   /'
            echo ""
        fi
        
        # IMDb
        IMDB_ID=$(echo "$RESULT" | jq -r '.imdb_id // ""' 2>/dev/null)
        if [ -n "$IMDB_ID" ] && [ "$IMDB_ID" != "" ] && [ "$IMDB_ID" != "null" ]; then
            echo -e "${YELLOW}🔗 IMDb:${NC} https://www.imdb.com/title/$IMDB_ID"
            echo ""
        fi
        
        # Trailer
        TRAILER_URL=$(echo "$RESULT" | jq -r '.trailer_url // ""' 2>/dev/null)
        if [ -n "$TRAILER_URL" ] && [ "$TRAILER_URL" != "" ] && [ "$TRAILER_URL" != "null" ]; then
            echo -e "${YELLOW}🎬 Bande-annonce:${NC} $TRAILER_URL"
            echo ""
        fi
        
        # Plateforme
        PLATFORM=$(echo "$RESULT" | jq -r '.platform // ""' 2>/dev/null)
        if [ -n "$PLATFORM" ] && [ "$PLATFORM" != "" ] && [ "$PLATFORM" != "null" ]; then
            echo -e "${YELLOW}📺 Plateforme:${NC} $PLATFORM"
            echo ""
        fi
        
        # Poster
        POSTER_URL=$(echo "$RESULT" | jq -r '.poster_url // ""' 2>/dev/null)
        if [ -n "$POSTER_URL" ] && [ "$POSTER_URL" != "" ] && [ "$POSTER_URL" != "null" ]; then
            echo -e "${YELLOW}🖼️  Affiche:${NC} $POSTER_URL"
            echo ""
        fi
        
        echo ""
    done
    
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Afficher le résultat le plus cohérent (premier résultat)
    echo -e "${YELLOW}🎯 RÉSULTAT LE PLUS COHÉRENT (recommandé):${NC}"
    FIRST_RESULT=$(echo "$SEARCH_RESPONSE" | jq '.results[0]' 2>/dev/null)
    FIRST_TITLE=$(echo "$FIRST_RESULT" | jq -r '.title // "N/A"' 2>/dev/null)
    FIRST_DIRECTOR=$(echo "$FIRST_RESULT" | jq -r '.director // ""' 2>/dev/null)
    FIRST_YEAR=$(echo "$FIRST_RESULT" | jq -r '.production_year // 0' 2>/dev/null)
    
    echo -e "${GREEN}   Titre:${NC} $FIRST_TITLE"
    if [ -n "$FIRST_DIRECTOR" ] && [ "$FIRST_DIRECTOR" != "" ]; then
        echo -e "${GREEN}   Réalisateur:${NC} $FIRST_DIRECTOR"
    fi
    if [ "$FIRST_YEAR" != "0" ] && [ "$FIRST_YEAR" != "null" ]; then
        echo -e "${GREEN}   Année:${NC} $FIRST_YEAR"
    fi
    echo ""
    
else
    echo -e "${RED}❌ Aucun résultat trouvé${NC}"
    echo -e "${YELLOW}💡 Vérifiez les logs avec: docker-compose logs -f backend | grep WebScraper${NC}"
    echo ""
    echo -e "${YELLOW}📋 Logs récents:${NC}"
    docker-compose logs --tail=30 backend 2>&1 | grep -E "WebScraper|Variantes|Google|AlloCiné|IMDb|Red One" | tail -10 || echo "Aucun log pertinent"
fi

echo ""
echo -e "${GREEN}✅ Test terminé${NC}"

