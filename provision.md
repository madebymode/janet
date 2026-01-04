# Provisioning: Database + Attachments

This imports the latest gzipped SQL backup and copies attachment files into the
production Docker volumes.

```bash
docker compose -f docker-compose.production.yml up -d postgres janet-init && \
LATEST_DB_BACKUP=$(ls -t janet_database_backups/*.gz | head -n 1) && \
gunzip -c "$LATEST_DB_BACKUP" | docker compose -f docker-compose.production.yml exec -T postgres psql -U janet -d janet && \
docker compose -f docker-compose.production.yml run --rm -v "$(pwd)/janet_attachments_backup:/host" janet-init sh -c "cp -a /host/. /var/lib/janet/attachments/"
```

Notes:
- Backups live in `janet_database_backups/` and should be `.gz` SQL exports.
- Attachments live in `janet_attachments_backup/` and will be copied into the `janet_attachments` volume.
- To import a specific backup, replace `LATEST_DB_BACKUP=...` with the file path.
