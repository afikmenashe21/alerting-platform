# Alert-Producer Documentation

This directory contains all documentation for the alert-producer service.

## 📖 Documentation Index

### Getting Started
- **[Getting Started Guide](./SETUP_AND_RUN.md)** - Complete setup and run instructions (recommended)

### Technical Documentation
- **[HTTP API Server](./API_SERVER.md)** - REST API for UI integration and job management
- **[Architecture & Structure](./STRUCTURE.md)** - Code organization, directory structure, and design decisions
- **[Event Structure](./EVENT_STRUCTURE.md)** - Complete alert event JSON schema and field specifications
- **[Partitioning Strategy](./PARTITIONING.md)** - How we use key-based partitioning to avoid hot partitions
- **[Testing Multiple Rules](./TESTING_MULTIPLE_RULES.md)** - Guide for testing multiple rules per client

## 🚀 Quick Links

- **Main README**: [../README.md](../README.md)
- **Makefile**: [../Makefile](../Makefile) - Build and run commands
- **Docker Compose**: [../../docker-compose.yml](../../docker-compose.yml) - Centralized infrastructure setup
