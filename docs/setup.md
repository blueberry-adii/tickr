# Tickr Setup (v2)

This document explains how to run Tickr locally in **two different ways**:

1. Run the Tickr server locally with MySQL & Redis (local or Docker)
2. Run everything using Docker Compose (recommended)

---

## Prerequisites

Make sure you have the following installed:

- Go (v1.25.5+ recommended)
- Docker & Docker Compose
- Git

---

## Clone the Repository

```bash
git clone https://github.com/blueberry-adii/tickr.git
cd tickr
```

Open the project in your editor:

```bash
code .
```

---

## Run Using Docker Compose (Recommended)

This is the simplest and recommended way to run Tickr locally.

It starts:

- Tickr server
- MySQL (with persistent volume)
- Redis

---

### 1. Start All Services

From the project root:

```bash
docker compose up --build
```

On first startup:

- MySQL database and tables are created using init.sql
- Redis starts empty (by design)
- Tickr initializes scheduler and workers

---

### 2. Verify Services

- Tickr API → http://localhost:8080
- Redis → localhost:6379
- MySQL → localhost:3306

---

### 3. Stopping the Stack

To stop services:

```bash
docker compose down
```

To stop and remove volumes (IMPORTANT: deletes MySQL data):

```
docker compose down -v
```

### 4. Running it without rebuilding

To run the build again without rebuilding:

```bash
docker compose up
```

OR

```bash
docker compose up -d
```

---

### Notes on Tickr v2 Architecture

- MySQL is the source of truth
- Redis is a disposable scheduling index
- Redis state loss is handled automatically
- Jobs overdue during downtime execute immediately after recovery
- In-flight jobs finish execution during graceful shutdown

---

Once the server is running and workers are active, you can start sending HTTP
requests to the API.

More details here -> [API Docs](./api.md)
