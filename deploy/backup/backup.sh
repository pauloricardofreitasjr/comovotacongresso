#!/bin/sh
# Backup lógico diário do Postgres, mantendo os últimos $BACKUP_KEEP (padrão 14).
# Variáveis: PGHOST, PGUSER, PGPASSWORD, PGDATABASE (as do libpq).
# A cópia FORA da máquina (outro provedor/bucket) depende da hospedagem: ver docs/operacao.md.
set -eu
DIR=${BACKUP_DIR:-/backups}
KEEP=${BACKUP_KEEP:-14}
INTERVALO=${BACKUP_INTERVALO_SEGUNDOS:-86400}

fazer() {
  arq="$DIR/comovota-$(date -u +%Y%m%dT%H%M%SZ).dump"
  # grava em .tmp e renomeia: um backup interrompido nunca parece válido
  if pg_dump -Fc -f "$arq.tmp" && pg_restore -l "$arq.tmp" >/dev/null && mv "$arq.tmp" "$arq"; then
    echo "backup ok: $arq ($(du -h "$arq" | cut -f1))"
    ls -1t "$DIR"/comovota-*.dump | tail -n +"$((KEEP + 1))" | xargs -r rm -f
  else
    rm -f "$arq.tmp"
    echo "backup FALHOU" >&2
    return 1
  fi
}

mkdir -p "$DIR"
if [ "${1:-}" = "uma-vez" ]; then fazer; exit $?; fi
while true; do
  fazer || true
  sleep "$INTERVALO"
done
