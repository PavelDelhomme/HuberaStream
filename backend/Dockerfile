FROM golang:1.21-alpine AS builder

# Installer les dépendances système nécessaires
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    ffmpeg-dev \
    ffmpeg

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum ./
RUN go mod download

# Copier le code source
COPY . .

# Construire l'application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /app/main ./cmd/server

FROM alpine:latest

# Installer FFmpeg pour le transcodage vidéo
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata

WORKDIR /app

# Copier le binaire depuis le builder
COPY --from=builder /app/main /app/main

# Créer les dossiers de stockage
RUN mkdir -p /app/storage/videos /app/storage/thumbnails

# Rendre le binaire exécutable
RUN chmod +x /app/main

EXPOSE 8080

CMD ["/app/main"]

