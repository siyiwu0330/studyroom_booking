# StudyRoom – Layered 

A simple **Gin + MongoDB + Redis** application implementing layered architecture for room booking and scheduling, served via **Nginx reverse proxy**.

---

##  Prerequisites
- Docker & Docker Compose
- `curl`, `jq`
- [`wrk`](https://github.com/wg/wrk) or use Docker image `williamyeh/wrk`

---

##  Run

```bash
# from the monolith project root
docker compose up -d --build
docker compose ps

