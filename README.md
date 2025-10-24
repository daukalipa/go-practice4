# Practice4 - Go + sqlx (Terminal CLI)

Simple project that connects to PostgreSQL using `sqlx`, performs CRUD operations,
and demonstrates a transactional balance transfer.

## Files
- `main.go` - single-file Go program (DB, CLI, models, transactions).
- `docker-compose.yml` - starts PostgreSQL (port 5430).
- `users.sql` - creates `users` table.
- `go.mod` - Go module file.

## Requirements
- Docker & Docker Compose
- Go 1.20+ (installed on your machine)
- (Optional) `psql` client

## Start PostgreSQL
From the project root (where `docker-compose.yml` and `users.sql` are):
```bash
docker-compose up -d
```

Verify the container is running:
```bash
docker ps
# look for the postgres container (should map host port 5430 -> container 5432)
```

## Create the `users` table
Copy `users.sql` into the container and run it:
```bash
docker cp users.sql go-practice4-postgres-1:/tmp/users.sql
docker exec -it go-practice4-postgres-1 psql -U user -d mydatabase -f /tmp/users.sql
```

Or run directly with the psql client:
```bash
PGPASSWORD=password psql -h localhost -p 5430 -U user -d mydatabase -f users.sql
```

> Tip: If you see `relation "users" already exists`, it means the table was created already. You can drop it first:
> ```bash
> docker exec -it go-practice4-postgres-1 psql -U user -d mydatabase -c "DROP TABLE IF EXISTS users;"
> ```

## Run the Go CLI
1. Fetch dependencies and run:
```bash
go mod tidy
go run main.go
```

2. After the program connects, you will see a prompt:
```
Practice4 CLI — type 'help' for commands
>
```

3. Common commands (type these at the `>` prompt):
- `help` - show help
- `list` - list all users
- `add <name> <email> <balance>` - add new user (no angle brackets)
    - Example: `add Alice alice@example.com 100`
- `get <id>` - show user by id
    - Example: `get 1`
- `transfer <fromID> <toID> <amount>` - transfer money between users
    - Example: `transfer 1 2 50`
- `exit` or `quit` - exit the CLI

## Example session
```
> add Alice alice@example.com 100
user added
> add Bob bob@example.com 10
user added
> list
1: Alice <alice@example.com> — 100.00
2: Bob <bob@example.com> — 10.00
> transfer 1 2 50
transfer ok
> list
1: Alice <alice@example.com> — 50.00
2: Bob <bob@example.com> — 60.00
> exit
```


