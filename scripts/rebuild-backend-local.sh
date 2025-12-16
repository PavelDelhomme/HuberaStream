#!/bin/bash

# Script pour compiler le backend localement et le copier dans le conteneur
# Utilisez ce script si docker build ne fonctionne pas

set -e

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}Compilation locale du backend...${NC}"

cd backend

# Vérifier que Go est installé
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go n'est pas installé. Installez Go pour continuer.${NC}"
    exit 1
fi

# Compiler le backend
echo -e "${GREEN}Compilation en cours...${NC}"
CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

if [ ! -f "main" ]; then
    echo -e "${RED}❌ Erreur lors de la compilation${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Compilation réussie${NC}"

# Copier le binaire dans le dossier de montage
echo -e "${GREEN}Copie du binaire dans le dossier de montage...${NC}"
cd ..

# Créer le dossier de montage s'il n'existe pas
mkdir -p .backend-bin

# Copier le binaire
cp backend/main .backend-bin/main
chmod +x .backend-bin/main

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Binaire copié avec succès dans /tmp/streammake-backend-bin/main${NC}"
    
    echo -e "${GREEN}Redémarrage du backend...${NC}"
    docker-compose restart backend
    sleep 8
    
    # Vérifier que le backend est démarré
    if docker-compose ps backend | grep -q "Up"; then
        echo -e "${GREEN}✅ Backend démarré avec le nouveau code${NC}"
    else
        echo -e "${RED}❌ Le backend n'a pas démarré correctement${NC}"
        echo -e "${YELLOW}Vérifiez les logs avec: docker-compose logs backend${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Erreur lors de la copie du binaire${NC}"
    exit 1
fi

# Nettoyer
rm -f backend/main

echo -e "${GREEN}✅ Terminé!${NC}"

