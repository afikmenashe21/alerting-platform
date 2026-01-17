# Rule Service UI

A simple React UI for managing clients, rules, and endpoints in the rule-service.

## Features

- **Clients Management**: Create, view, and list clients
- **Rules Management**: Full CRUD operations for alerting rules
- **Endpoints Management**: Create, update, delete, and toggle endpoints for rules

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- rule-service running on http://localhost:8081

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The app will be available at http://localhost:3000

### Build

```bash
npm run build
```

## API Configuration

The UI connects to the rule-service API at `http://localhost:8081`. This is configured via Vite proxy in `vite.config.js`.
