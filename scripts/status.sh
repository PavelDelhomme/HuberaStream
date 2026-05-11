#!/bin/bash

# Couleurs
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="streammake"

echo -e "${GREEN}📊 Statut détaillé des services StreamMake${NC}"
echo "=========================================="
echo ""

# Fonction pour obtenir le statut d'un service
get_service_status() {
    local service_name=$1
    local container_name="${PROJECT_NAME}-${service_name}"
    
    # Vérifier si le conteneur existe et est en cours d'exécution
    if ! docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
        echo -e "  ${RED}❌ DOWN ${service_name}${NC}"
        return
    fi
    
    # Obtenir le statut de santé
    local health=$(docker inspect --format='{{.State.Health.Status}}' "${container_name}" 2>/dev/null || echo "none")
    
    # Obtenir tous les ports mappés
    local all_ports=$(docker port "${container_name}" 2>/dev/null)
    
    # Extraire le port principal (premier port TCP non-privilégié)
    local ports=""
    if [ -n "$all_ports" ]; then
        # Pour chaque ligne de port, extraire host:container
        while IFS= read -r line; do
            if [ -n "$line" ]; then
                local container_port=$(echo "$line" | awk '{print $1}' | cut -d'/' -f1)
                local host_info=$(echo "$line" | awk '{print $3}')
                local host_port=$(echo "$host_info" | cut -d':' -f2)
                
                # Ignorer les ports de debug (2345) et privilégiés (< 1024 sauf 80, 443)
                if [ "$host_port" != "2345" ] && [ -n "$host_port" ] && [ -n "$container_port" ]; then
                    # Prioriser les ports HTTP/API (80, 8080, 3000, 3300-3399)
                    if [ "$host_port" -ge 3300 ] && [ "$host_port" -le 3399 ] || [ "$container_port" = "80" ] || [ "$container_port" = "8080" ] || [ "$container_port" = "3000" ]; then
                        ports="${host_port} → ${container_port}"
                        break
                    fi
                fi
            fi
        done <<< "$all_ports"
        
        # Si pas de port principal trouvé, prendre le premier
        if [ -z "$ports" ]; then
            local first_line=$(echo "$all_ports" | head -1)
            if [ -n "$first_line" ]; then
                local container_port=$(echo "$first_line" | awk '{print $1}' | cut -d'/' -f1)
                local host_info=$(echo "$first_line" | awk '{print $3}')
                local host_port=$(echo "$host_info" | cut -d':' -f2)
                if [ -n "$host_port" ] && [ -n "$container_port" ]; then
                    ports="${host_port} → ${container_port}"
                fi
            fi
        fi
    fi
    
    # Obtenir le temps de fonctionnement
    local started=$(docker inspect --format='{{.State.StartedAt}}' "${container_name}" 2>/dev/null)
    local minutes="0"
    if [ -n "$started" ]; then
        if command -v date >/dev/null 2>&1; then
            local started_epoch=$(date -d "$started" +"%s" 2>/dev/null || echo "0")
            local now_epoch=$(date +"%s")
            if [ "$started_epoch" != "0" ]; then
                local diff=$((now_epoch - started_epoch))
                minutes=$((diff / 60))
            fi
        fi
    fi
    
    # Déterminer l'icône et la couleur selon le statut
    local status_icon=""
    local status_text=""
    if [ "$health" = "healthy" ]; then
        status_icon="${GREEN}✅"
        status_text="${GREEN}healthy${NC}"
    elif [ "$health" = "unhealthy" ]; then
        status_icon="${RED}⚠️"
        status_text="${RED}unhealthy${NC}"
    elif [ "$health" = "starting" ]; then
        status_icon="${YELLOW}🔄"
        status_text="${YELLOW}starting${NC}"
    else
        status_icon="${GREEN}✅"
        status_text=""
    fi
    
    # Afficher le statut avec printf pour l'alignement
    local service_display=$(printf "%-30s" "${service_name}")
    local ports_display=$(printf "%-20s" "${ports}")
    
    if [ -n "$status_text" ]; then
        echo -e "  ${status_icon} UP   ${service_display} ${ports_display} [${status_text}] (${minutes} minutes)"
    else
        echo -e "  ${status_icon} UP   ${service_display} ${ports_display} (${minutes} minutes)"
    fi
}

# Services par catégorie
echo -e "${CYAN}🏗️  Infrastructure:${NC}"
echo ""
get_service_status "postgres"
get_service_status "redis"
echo ""

echo -e "${CYAN}🌐 Frontend & Gateway:${NC}"
echo ""
get_service_status "frontend-nextjs"
get_service_status "nginx"
echo ""

echo -e "${CYAN}🔧 Backend:${NC}"
echo ""
get_service_status "backend"
echo ""

echo -e "${CYAN}📥 Clients Torrent:${NC}"
echo ""
get_service_status "transmission"
get_service_status "qbittorrent"
get_service_status "deluge"
get_service_status "rtorrent"
echo ""

# Compter les services actifs
total_services=$(docker ps --format "{{.Names}}" | grep "^${PROJECT_NAME}-" | wc -l)
echo -e "${GREEN}📊 Résumé : ${total_services} service(s) actif(s)${NC}"
echo ""
