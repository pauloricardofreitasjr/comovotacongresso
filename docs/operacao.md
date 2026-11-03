# Operação

Roteiro para publicar e manter o site em **uma máquina com Docker** (VPS). Os arquivos estão em `deploy/`. A escolha da hospedagem continua em aberto; nada aqui depende de um provedor específico, exceto o que está marcado.

## Como a produção é montada

`deploy/docker-compose.prod.yml` sobe:

| Serviço | Função |
|---|---|
| `db` | PostgreSQL 16, sem porta publicada |
| `migrate` | aplica as migrations antes de `api` e `etl` |
| `api` | API REST, só na rede interna |
| `web` | site Next.js; consome a API pelo servidor, então a API **não precisa ser pública** |
| `etl` | `etl agendar`: `daily` a cada `DAILY_EVERY` (2h) e `sync-parlamentares` aos domingos 04:00 (Brasília) |
| `caddy` | únicas portas públicas (80/443), HTTPS automático, redireciona o apex para `www`, rate limiting por IP (120 req/min) |
| `backup` | `pg_dump` diário, mantém os 14 últimos no volume `backups` |

## Primeiro deploy

1. Aponte o DNS (`A`/`AAAA`) de `comovotacongresso.com.br` e `www.comovotacongresso.com.br` para a máquina. As portas 80 e 443 precisam estar abertas: o Caddy obtém o certificado sozinho.
2. `cp deploy/.env.prod.example deploy/.env.prod` e preencha. Gere a senha com `openssl rand -hex 24` (hex, porque ela entra numa URL).
3. Suba:
   ```bash
   docker compose --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml up -d --build
   ```
4. Carga inicial (demora, respeita o intervalo mínimo entre requisições à Câmara):
   ```bash
   docker compose --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml run --rm etl /app/etl backfill
   ```
5. Confira `https://www.comovotacongresso.com.br`, o redirecionamento do apex e `docker compose ... ps` (o `etl` deve ficar `healthy`).

## Alertas

- **Falha de job:** defina `ALERT_WEBHOOK_URL` (Slack, Discord ou Mattermost: o corpo traz `text` e `content`). O alerta sai na primeira falha de uma sequência e quando o job se recupera, não a cada tentativa.
- **Daily parado:** o healthcheck do `etl` roda `etl saude -max 6h`, que falha se o último `daily` bem-sucedido for mais antigo. Isso cobre o caso em que o próprio agendador morre e não consegue avisar. **Isso só aparece em `docker ps`**; para ser avisado, use um monitor externo (ex.: UptimeRobot no site) e/ou um cron do host que consulte `docker inspect`. Não incluído.
- Um `daily` com qualquer votação ou sessão com erro conta como falha (os detalhes ficam em `etl_errors`).

## Backup e restauração

Os dumps ficam no volume `backups`, **na mesma máquina do banco**. Isso protege contra erro humano, não contra perda da máquina: copie-os para fora (outro provedor ou bucket). O comando depende do destino e não está automatizado.

Backup imediato:
```bash
docker compose --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml exec backup /bin/sh /backup.sh uma-vez
```

Restauração (testada: dump e restauração em banco novo deram as mesmas contagens, 436.085 votos e 648 parlamentares):
```bash
# 1. copie o dump para o container do banco (ou monte o volume) e crie um banco limpo
docker compose ... exec db psql -U comovota -d postgres -c "CREATE DATABASE restaurado"
docker compose ... exec db pg_restore -U comovota -d restaurado --no-owner /caminho/comovota-AAAAMMDDTHHMMSSZ.dump
# 2. confira as contagens; só então troque o banco ativo (renomeie ou ajuste a URL)
```
Os dados são todos reconstruíveis a partir da Câmara (`etl backfill`), então o backup acelera a recuperação em vez de ser a única cópia.

## Rate limiting

Configurado no Caddy (`deploy/Caddyfile`, módulo `caddy-ratelimit`, imagem própria em `deploy/caddy/`). Testado: com limite de 3 eventos, a 4ª requisição recebeu `429`. **Se houver CDN ou proxy na frente (Cloudflare etc.)**, a chave `{remote_host}` passa a ser o IP da CDN e todos os visitantes dividem o mesmo limite: troque pelo cabeçalho do IP real do cliente e aceite conexões só dos IPs da CDN. O valor de 120/min é um chute; ajuste com o tráfego real.

## Antes de publicar

- Senhas: só `deploy/.env.prod` (ignorado pelo git). As senhas do `docker-compose.yml` da raiz são de desenvolvimento.
- Pendências do responsável que bloqueiam o lançamento estão em `docs/pendencias.md` (licença das fotos, metodologia, indexação).
