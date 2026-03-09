include .envrc

.PHONY: default
default:
	go run ./cmd/api

.PHONY: help
help:
	@echo "usage"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n "Are u sure [y/N]" && read ans && [ $${ans:-N} = y ]

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api

run/build:
	go build ./cmd/api
## run/help: provide config details for cmd/api application
.PHONY: run/help
run/help:
	go run ./cmd/api --help

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	pgcli ${SCOUTER_DB_DSN}

## db/migrations/up: apply all up database migrations
.PHONY: db/migration/up
db/migration/up: confirm
	@echo "Running migrations"
	migrate -path ./migrations/ -database ${SCOUTER_DB_DSN} up 


## db/migrations/down : apply all down database migration
.PHONY: db/migration/down
db/migration/down: confirm
	@echo "Running migrations"
	migrate -path ./migrations/ -database ${SCOUTER_DB_DSN} down

## db/migrations/new v=$1: force to version v if dirty found
.PHONY: db/migration/force
db/migration/force: confirm
	@migrate -path ./migrations/ -database ${SCOUTER_DB_DSN} force ${v}
