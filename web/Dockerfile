# Contexto de build: a raiz do repositório (docker build -f web/Dockerfile .),
# pois a metodologia é lida de docs/metodologia.md.
FROM node:22-alpine AS deps
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY web/ .
RUN npm run build

FROM node:22-alpine AS run
ENV NODE_ENV=production PORT=3000 HOSTNAME=0.0.0.0 METODOLOGIA_PATH=/app/metodologia.md
WORKDIR /app
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY docs/metodologia.md /app/metodologia.md
# O Next grava o cache de dados (fetch com revalidate) em .next/cache; o usuário node precisa poder criá-lo.
RUN mkdir -p .next/cache && chown -R node:node /app
USER node
EXPOSE 3000
CMD ["node", "server.js"]
