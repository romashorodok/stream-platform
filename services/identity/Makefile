
up:
	migrate '-database=postgres://user:password@localhost:5433/postgres?sslmode=disable' '-source=file://./migrations' up

down:
	migrate '-database=postgres://user:password@localhost:5433/postgres?sslmode=disable' '-source=file://./migrations' down -all

generate:
	jet '-dsn=postgres://user:password@localhost:5433/postgres?sslmode=disable' -path=./internal/storage/schema


# migrate create -ext sql -dir ./migrations -seq create_users_table
# migrate '-database=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-source=file://./migrations' force 1
# migrate '-database=postgres://user:password@localhost:5432/postgres?sslmode=disable' '-source=file://./migrations' up
