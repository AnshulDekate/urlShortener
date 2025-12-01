-------------------------------------------
Setup: 

git clone github.com/AnshulDekate/urlShortener.git
go mod tidy
docker-compose up --build

-------------------------------------------
Curls to test: 

HEALTCHECK
curl --location 'http://127.0.0.1:8080/healthcheck'

SHORTEN_URL
curl --location 'http://127.0.0.1:8080/shorten' \
--header 'Content-Type: application/json' \
--data '{
    "long_url": "https://poltora.dev/rust-vs-go-memory/"
}'

PAGINATION
curl --location 'http://127.0.0.1:8080/urls?page=1&limit=5'
--------------------------------------------

To inspect the db directly:
docker compose exec db psql -U anshul -d anshul




