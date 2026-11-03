# Caddy com o módulo de rate limiting (não vem na imagem oficial).
FROM caddy:2.8-builder AS build
RUN xcaddy build --with github.com/mholt/caddy-ratelimit

FROM caddy:2.8-alpine
COPY --from=build /usr/bin/caddy /usr/bin/caddy
