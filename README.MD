# URL Shortener

## Setup

Clone and start the project:

```bash
git clone https://github.com/AnshulDekate/urlShortener.git
cd urlShortener
go mod tidy
docker compose up --build
```

The service listens on port 8080 by default.

## Endpoints / Example curls

Healthcheck:

```bash
curl --location 'http://127.0.0.1:8080/healthcheck'
```

Shorten a URL:

```bash
curl --location 'http://127.0.0.1:8080/shorten' \
  --header 'Content-Type: application/json' \
  --data '{
    "long_url": "https://poltora.dev/rust-vs-go-memory/"
  }'
```

List shortened URLs (pagination):

```bash
curl --location 'http://127.0.0.1:8080/urls?page=1&limit=5'
```

## Inspect the database

Open a psql shell in the running DB container (macOS / Linux):

```bash
docker compose exec db psql -U anshul -d anshul
```
## Key Design Decisions

### Short URL Generation Flow
1. **Validate & parse** input long URL
2. **Idempotency check** — find existing short code for this URL
3. **Insert long URL** into DB, get auto-increment ID
4. **Generate random base62 code** (configurable length, default 10 chars)
5. **Check uniqueness** — retry on collision (up to 5 times)
6. **Update DB row** with final short code
7. **Return** short code to client

**Design Choice: Random codes (not sequential)**
- **Pro:** Non-predictable, harder to enumerate URLs
- **Con:** Risk of collisions (mitigated by retry loop)

### Rate Limiting
- 20 requests per minute (configurable)

### Not Implemented
- Authentication
- Caching
- Analytics

### Deployment
- Docker Compose 