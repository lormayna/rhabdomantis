# Rhabdomantis

A specialized tool for discovering, scanning, and verifying Ollama LLM instances exposed on the internet.

## Project Overview

Rhabdomantis is a Go-based CLI application that automates the lifecycle of LLM instance discovery:
1.  **Sync:** Discovers potential Ollama hosts using the Shodan API.
2.  **Scan:** Probes discovered hosts to retrieve available models and metadata via the Ollama API (`/api/tags`).
3.  **Verify:** Validates the functional status of discovered models by performing test inferences (e.g., simple arithmetic) and recording the results.

### Core Technologies
- **Language:** Go 1.26.1
- **CLI Framework:** [urfave/cli/v2](https://github.com/urfave/cli/v2)
- **Database:** SQLite (managed via [sqlc](https://sqlc.dev/))
- **External APIs:** 
    - [Shodan](https://www.shodan.io/) (for discovery)
    - [Ollama API](https://ollama.com/) (for scanning and verification)
- **Concurrency:** [errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) for parallel processing of hosts.

## Project Structure

- `main.go`: Application entry point and CLI command definitions.
- `cmd/`: Implementation of core commands.
    - `sync.go`: Shodan integration for host discovery.
    - `scan.go`: Ollama API probing and model metadata collection.
    - `verify.go`: Inference testing and functional validation.
- `db/`: Database layer.
    - `schema.sql`: SQLite schema definition.
    - `queries.sql`: SQL queries used by `sqlc`.
    - `*.sql.go`: Generated Go code for database access.
- `models/`: Common data structures and API response models.
- `sqlc.yaml`: Configuration for the `sqlc` code generator.

## Building and Running

### Prerequisites
- Go 1.26+ installed.
- A Shodan API key (currently hardcoded in `main.go`, but should ideally be provided via environment variable `SHODAN_API_KEY`).

### Build
```bash
go build -o rhabdomantis main.go
```

### Commands
- **Sync:** Fetch new hosts from Shodan.
  ```bash
  ./rhabdomantis sync
  ```
- **Scan:** Probe hosts for available models.
  ```bash
  ./rhabdomantis scan
  ```
- **Verify:** Perform inference tests on discovered models.
  ```bash
  ./rhabdomantis verify
  ```
- **Export:** Export uncensored models to a structured JSON file.
  ```bash
  ./rhabdomantis export -n 5
  ```

## Development Conventions

- **Database Management:** Do not edit `db/db.go` or `db/models.go` manually. Update `db/schema.sql` or `db/queries.sql` and run `sqlc generate`.
- **Concurrency:** The application uses a worker pool pattern via `errgroup`. The default number of workers is 3 (defined in `main.go`).
- **Error Handling:** Use `log/slog` for structured logging.
- **API Keys:** Avoid hardcoding secrets. Ensure `SHODAN_API_KEY` is handled securely.

## TODO / Future Improvements
- [ ] Move Shodan API key to environment variables or a configuration file.
- [ ] Add support for custom prompts in the `verify` command.
- [ ] Implement export functionality for discovered data.
