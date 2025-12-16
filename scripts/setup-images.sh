#!/bin/bash

# Script pour créer les images Docker nécessaires sans utiliser buildx
# Usage: ./scripts/setup-images.sh

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🔨 Création des images Docker nécessaires...${NC}"
echo ""

# 1. Backend
echo -e "${YELLOW}[1/2] Création de l'image backend...${NC}"
if [ ! -f .backend-bin/main ]; then
    echo -e "${YELLOW}   Compilation du backend (statique pour Alpine/musl)...${NC}"
    cd backend
    # Compiler de manière vraiment statique pour Alpine (musl)
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags='-s -w -linkmode external -extldflags "-static"' -tags netgo -o ../.backend-bin/main ./cmd/server || {
        # Si la compilation statique échoue, essayer sans les flags externes
        echo -e "${YELLOW}   Tentative de compilation sans flags externes...${NC}"
        CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-s -w' -tags netgo -o ../.backend-bin/main ./cmd/server
    }
    cd ..
fi

if [ -f .backend-bin/main ]; then
    echo -e "${YELLOW}   Création du Dockerfile minimal...${NC}"
    mkdir -p .backend-bin
    cat > .backend-bin/Dockerfile << 'EOF'
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata ffmpeg
COPY main /app/main
RUN chmod +x /app/main
WORKDIR /app
RUN mkdir -p /app/storage/videos /app/storage/thumbnails
CMD ["/app/main"]
EOF
    
    echo -e "${YELLOW}   Construction de l'image...${NC}"
    cd .backend-bin
    DOCKER_BUILDKIT=0 docker build -t streammake-backend:latest . || {
        echo -e "${RED}❌ Erreur lors de la construction de l'image backend${NC}"
        echo -e "${YELLOW}💡 Essayez de mettre à jour Docker ou utilisez: make rebuild-backend-local${NC}"
        exit 1
    }
    cd ..
    echo -e "${GREEN}✅ Image backend créée${NC}"
else
    echo -e "${RED}❌ Binaire backend non trouvé${NC}"
    exit 1
fi

echo ""

# 2. Frontend
echo -e "${YELLOW}[2/2] Création de l'image frontend...${NC}"
if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "streammake-frontend:latest" 2>/dev/null; then
    echo -e "${GREEN}✅ Image frontend déjà présente${NC}"
else
    echo -e "${YELLOW}   Construction de l'image frontend...${NC}"
    cd frontend
    
    # Créer un Dockerfile temporaire si nécessaire
    if [ ! -f Dockerfile ]; then
        cat > Dockerfile << 'EOF'
FROM node:20-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["npm", "run", "dev"]
EOF
    fi
    
    # Essayer de construire avec DOCKER_BUILDKIT=0
    if DOCKER_BUILDKIT=0 docker build -t streammake-frontend:latest . 2>&1; then
        echo -e "${GREEN}✅ Image frontend créée${NC}"
    else
        echo -e "${YELLOW}⚠️  Construction frontend échouée, création d'une image minimale...${NC}"
        cd ..
        mkdir -p .frontend-bin
        cd .frontend-bin
        # Créer une image minimale basée sur node:20-alpine
        cat > Dockerfile << 'EOF'
FROM node:20-alpine
WORKDIR /app
RUN npm install -g vite@latest
EXPOSE 3000
CMD ["sh", "-c", "npm install && npm run dev"]
EOF
        if DOCKER_BUILDKIT=0 docker build -t streammake-frontend:latest . 2>&1; then
            echo -e "${GREEN}✅ Image frontend minimale créée${NC}"
        else
            echo -e "${RED}❌ Impossible de créer l'image frontend${NC}"
            echo -e "${YELLOW}💡 Le frontend utilisera l'image node:20-alpine directement${NC}"
            echo -e "${YELLOW}⚠️  Vous devrez peut-être modifier docker-compose.yml pour utiliser node:20-alpine${NC}"
            cd ..
            exit 0  # Ne pas échouer, on peut continuer sans image frontend personnalisée
        fi
        cd ..
    fi
    cd ..
    
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "streammake-frontend:latest" 2>/dev/null; then
        echo -e "${GREEN}✅ Image frontend créée${NC}"
    else
        echo -e "${YELLOW}⚠️  Image frontend non créée${NC}"
    fi
fi

echo ""
echo -e "${GREEN}✅ Images Docker prêtes !${NC}"
echo -e "${YELLOW}💡 Vous pouvez maintenant démarrer les services avec:${NC}"
echo -e "${GREEN}   docker compose up -d --no-build${NC}"
echo -e "${GREEN}   ou${NC}"
echo -e "${GREEN}   make up${NC}"

