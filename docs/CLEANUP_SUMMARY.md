# Service Cleanup Summary

## Files Removed

### Redundant Infrastructure Files
- ✅ Removed 6x `docker-compose.yml` files from services:
  - `services/alert-producer/docker-compose.yml`
  - `services/rule-service/docker-compose.yml`
  - `services/rule-updater/docker-compose.yml`
  - `services/evaluator/docker-compose.yml`
  - `services/aggregator/docker-compose.yml`
  - `services/sender/docker-compose.yml`

- ✅ Removed 1x `.dockerignore` file:
  - `services/alert-producer/.dockerignore`

**Total: 7 files removed**

## Files Kept (Not Redundant)

### Migrations
- ✅ **KEPT**: All migration files in `services/*/migrations/`
  - These are required by the centralized migration runner
  - The runner reads from service directories: `scripts/migrations/run-migrations.sh` searches `services/*/migrations/`

### Service Scripts
- ✅ **KEPT**: All `services/*/scripts/run-all.sh` files
  - These verify dependencies and run individual services
  - Still useful for development and testing

### Service Makefiles
- ✅ **KEPT**: All `services/*/Makefile` files
  - Service-specific build and run commands

### Service Documentation
- ✅ **KEPT**: Service-specific READMEs and documentation
  - Each service has unique setup and usage instructions

## Documentation Updates

Updated all references to `docker-compose.yml` in service documentation:
- `services/alert-producer/docs/*.md` - Updated 7 files
- `services/sender/README.md`
- `services/evaluator/README.md`
- `services/rule-updater/README.md`
- `services/rule-service/README.md`

All references now point to the root-level `docker-compose.yml` and clarify that infrastructure is centralized.

## Verification

```bash
# Verify no docker-compose.yml files remain in services
find services -name "docker-compose.yml" -o -name ".dockerignore"
# Should return: (empty - no files found)
```

## Result

- **Cleaner structure**: Services no longer have redundant infrastructure definitions
- **Single source of truth**: All infrastructure defined at root level
- **Updated documentation**: All references point to centralized infrastructure
- **No breaking changes**: Migrations and scripts remain functional
