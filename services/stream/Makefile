
migrate:
	go run ./cmd/migrate/main.go '-db=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-migrations=file://./migrations'

generate:
	jet '-dsn=postgres://user:password@localhost:5432/postgres?sslmode=disable' -path=./internal/storage/schema

up:
	migrate '-database=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-source=file://./migrations' up

down:
	migrate '-database=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-source=file://./migrations' down -all

# migrate create -ext sql -dir ./migrations -seq create_users_table
# migrate '-database=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-source=file://./migrations' force 1
# migrate '-database=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-source=file://./migrations' up
