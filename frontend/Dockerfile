FROM node:20-alpine

WORKDIR /app

# Copier les fichiers de dépendances
COPY package.json package-lock.json* ./

# Installer les dépendances
RUN npm ci

# Copier le code source
COPY . .

EXPOSE 3000

# Mode développement avec Vite
CMD ["npm", "run", "dev"]

