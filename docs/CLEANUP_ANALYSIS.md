# Service Cleanup Analysis

This document identifies redundant files and code across services after centralizing infrastructure.

## ✅ Files to KEEP (Not Redundant)

### Migrations (`services/*/migrations/`)
**Status: KEEP** - These are required!
- The centralized migration runner (`scripts/migrations/run-migrations.sh`) reads from `services/*/migrations/` directories
- Migrations are organized by service but run centrally
- Location: `services/rule-service/migrations/`, `services/aggregator/migrations/`

### Service Scripts (`services/*/scripts/run-all.sh`)
**Status: KEEP** - These are still useful!
- Each service's `run-all.sh` verifies dependencies and runs the individual service
- They use centralized infrastructure but provide service-specific setup
- Useful for running services individually during development

### Service Makefiles (`services/*/Makefile`)
**Status: KEEP** - Service-specific build commands
- Each service has its own Makefile for building and running
- Provides service-specific targets

## ❌ Files to REMOVE (Redundant)

### docker-compose.yml in Services
**Status: REMOVE** - Completely redundant!
- All services have `docker-compose.yml` files that define infrastructure
- Infrastructure is now centralized at root level (`docker-compose.yml`)
- These service-level files are no longer used
- **Files to remove:**
  - `services/alert-producer/docker-compose.yml`
  - `services/rule-service/docker-compose.yml`
  - `services/rule-updater/docker-compose.yml`
  - `services/evaluator/docker-compose.yml`
  - `services/aggregator/docker-compose.yml`
  - `services/sender/docker-compose.yml`

### .dockerignore in alert-producer
**Status: REMOVE** - Not needed if service isn't dockerized
- Only `alert-producer` has a `.dockerignore` file
- If services aren't being built as Docker images, this is redundant
- **File to remove:** `services/alert-producer/.dockerignore`

## ⚠️ Files to REVIEW (Potentially Redundant)

### Service Documentation
**Status: REVIEW** - Some may be redundant
- `services/alert-producer/docs/` - Extensive documentation that might overlap with root docs
- Individual service READMEs are useful and should be kept
- Service-specific troubleshooting docs are useful

### Memory Bank Files
**Status: KEEP** - Service-specific design decisions
- Each service has its own `memory-bank/` directory
- These capture service-specific patterns and decisions
- Should be kept for service-level context

## Summary

**Files to Remove:**
- 6x `docker-compose.yml` files (one per service)
- 1x `.dockerignore` file (alert-producer)

**Total: 7 files to remove**
