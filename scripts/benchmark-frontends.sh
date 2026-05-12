#!/usr/bin/env bash
# Mesures : tailles disque + durée de build (Next vs Vite legacy).
#
# Sur ta machine (npm local) :
#   bash scripts/benchmark-frontends.sh
#
# Tout dans Docker (pas besoin de Node/npm installés sur l’hôte) :
#   USE_DOCKER=1 bash scripts/benchmark-frontends.sh
#
# Avant USE_DOCKER=1, construire l’image Next au moins une fois :
#   docker compose -f docker-compose.yml -p streammake build frontend-nextjs

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT/docker-compose.yml}"
PROJECT_NAME="${PROJECT_NAME:-streammake}"

echo "========== StreamMake — benchmark frontends =========="
echo "ROOT=$ROOT"
if [[ "${USE_DOCKER:-0}" == "1" ]]; then
  echo "Mode: USE_DOCKER=1 (builds dans les conteneurs / Docker run)"
else
  echo "Mode: local (npm sur l’hôte)"
fi
echo ""

section() {
  echo ""
  echo "---- $1 ----"
}

compose() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" "$@"
  else
    echo "Erreur: docker compose introuvable."
    exit 1
  fi
}

run_next_in_docker() {
  section "frontend-nextjs (Next.js) — Docker"
  echo "docker compose run frontend-nextjs … (montage ./frontend-nextjs → /app)"
  compose build frontend-nextjs 2>/dev/null || true
  compose run --rm -T --no-deps \
    -e CI=1 \
    -e NODE_ENV=production \
    -e NEXT_TELEMETRY_DISABLED=1 \
    frontend-nextjs \
    sh -c 'set -e; cd /app; echo "du avant build:"; du -sh node_modules 2>/dev/null || true; du -sh .next 2>/dev/null || true; if [ ! -f node_modules/.bin/next ]; then echo "npm ci…"; npm ci --legacy-peer-deps; fi; echo "npm run build (NODE_ENV=production requis pour next build)…"; START=$(date +%s); npm run build; END=$(date +%s); echo "Durée build Next : $((END - START)) s"; du -sh .next'
}

run_vite_in_docker() {
  section "frontend (Vite legacy) — Docker (image node:20-alpine, volume ./frontend)"
  echo "docker run node:20-alpine … (pas de service compose pour ce legacy)"
  docker run --rm \
    -v "$ROOT/frontend:/app" \
    -w /app \
    -e CI=1 \
    node:20-alpine \
    sh -c 'set -e; echo "du avant:"; du -sh node_modules 2>/dev/null || true; du -sh dist 2>/dev/null || true; if [ ! -f node_modules/.bin/vite ]; then if [ -f package-lock.json ]; then echo "npm ci…"; npm ci; else echo "npm install (pas de package-lock.json dans ./frontend)…"; npm install; fi; fi; echo "npm run build (vite)…"; START=$(date +%s); npm run build; END=$(date +%s); echo "Durée build Vite : $((END - START)) s"; du -sh dist'
}

run_next_local() {
  section "frontend-nextjs (Next.js)"
  cd "$ROOT/frontend-nextjs"
  if [[ ! -f node_modules/.bin/next ]]; then
    echo "npm ci…"
    npm ci --legacy-peer-deps
  fi
  echo "du node_modules / .next (avant build complet) :"
  du -sh node_modules 2>/dev/null || true
  du -sh .next 2>/dev/null || true
  echo "npm run build…"
  START=$(date +%s)
  npm run build
  END=$(date +%s)
  echo "Durée build Next : $((END - START)) s"
  du -sh .next
}

run_vite_local() {
  section "frontend (Vite + React legacy)"
  cd "$ROOT/frontend"
  if [[ ! -f node_modules/.bin/vite ]]; then
    if [[ -f package-lock.json ]]; then
      echo "npm ci…"
      npm ci
    else
      echo "npm install (pas de package-lock.json)…"
      npm install
    fi
  fi
  echo "du node_modules / dist :"
  du -sh node_modules 2>/dev/null || true
  du -sh dist 2>/dev/null || true
  echo "npm run build…"
  START=$(date +%s)
  npm run build
  END=$(date +%s)
  echo "Durée build Vite : $((END - START)) s"
  du -sh dist
}

if [[ "${USE_DOCKER:-0}" == "1" ]]; then
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker est requis pour USE_DOCKER=1."
    exit 1
  fi
  run_next_in_docker
  run_vite_in_docker
else
  if ! command -v npm >/dev/null 2>&1; then
    echo "npm introuvable. Lance avec Docker :"
    echo "  USE_DOCKER=1 bash scripts/benchmark-frontends.sh"
    exit 1
  fi
  run_next_local
  run_vite_local
fi

echo ""
echo "──────── Résumé ────────"
echo "• Next.js : durée « Durée build Next », taille .next en fin de section Docker/local."
echo "• Vite legacy : durée « Durée build Vite », taille dist."
if [[ "${USE_DOCKER:-0}" == "1" ]]; then
  echo "• Docker : .next et node_modules Next peuvent être dans le volume anonyme monté sur /app ; ./frontend-nextjs et ./frontend reflètent le code et dist sur l’hôte."
  echo "• Build Next en CI : garder NODE_ENV=production pour next build (déjà forcé ici en USE_DOCKER=1)."
fi
echo ""
echo "Terminé."
