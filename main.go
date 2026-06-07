package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/lormayna/rhabdomantis/cmd"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v2"
)

//go:embed db/migrations/*.sql
var embedMigrations embed.FS

const workers = 10

func readConfig() (*cmd.Config, error) {
	// Carica variabili d'ambiente da .env se presente
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	var conf cmd.Config
	if err := env.Parse(&conf); err != nil {
		return nil, fmt.Errorf("error parsing environment variables: %w", err)
	}
	return &conf, nil
}

func main() {
	slog.Info("Starting Rhabdomantis...")
	conf, err := readConfig()
	if err != nil {
		slog.Error("Errore nella lettura della configurazione", "error", err)
		os.Exit(1)
	}
	if conf.ShodanAPIKey == "" {
		fmt.Println("Error: SHODAN_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	// Apri connessione al database SQLite
	dbConn, err := sql.Open("sqlite3", conf.DBFile)
	if err != nil {
		slog.Error("Errore nell'apertura del database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// Esegui migrazioni con goose
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		slog.Error("Errore nell'impostazione del dialetto goose", "error", err)
		os.Exit(1)
	}

	if err := goose.Up(dbConn, "db/migrations"); err != nil {
		slog.Error("Errore durante l'esecuzione delle migrazioni", "error", err)
		os.Exit(1)
	}

	app := &cli.App{
		Name:  "rhabdomantis",
		Usage: "Tool for scanning LLMs",
		Commands: []*cli.Command{
			{
				Name:    "scan",
				Aliases: []string{"s"},
				Usage:   "Start a new scan",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Value:   ".",
						Usage:   "P",
					},
				},
				Action: func(c *cli.Context) error {
					path := c.String("path")
					fmt.Printf("🔍 Start scan in: %s...\n", path)

					err := cmd.Scan(conf)
					if err != nil {
						slog.Error("Errore durante la scansione", "error", err)
						return err
					}
					fmt.Println("✅ Scan completed successfully!")
					return nil
				}, // Chiusura corretta della funzione Action
			}, // Chiusura corretta del comando scan
			{
				Name:    "sync",
				Aliases: []string{"y"},
				Usage:   "Retrieve data from Shodan and update the database",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Forza la sincronizzazione anche se i dati sono aggiornati",
					},
				},
				Action: func(c *cli.Context) error {
					fmt.Println("♻️  Sincronizzazione standard in corso...")
					err := cmd.Sync(conf)
					if err != nil {
						slog.Error("Errore durante la sincronizzazione", "error", err)
						return err
					}
					fmt.Println("✅ Sincronizzazione completata con successo!")
					return nil
				},
			},
			{
				Name:    "verify",
				Aliases: []string{"v"},
				Usage:   "Verify the presence of LLMs on active hosts",
				Action: func(c *cli.Context) error {
					fmt.Printf("🔍 Start verification")

					err := cmd.Verify(conf)
					if err != nil {
						slog.Error("Errore durante la verifica", "error", err)
						return err
					}
					fmt.Println("✅ Verification completed successfully!")
					return nil
				}, // Chiusura corretta della funzione Action
			}, // Chiusura corretta del comando scan
			{
				Name:    "export",
				Aliases: []string{"e"},
				Usage:   "Export uncensored models to JSON",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "num",
						Aliases: []string{"n"},
						Value:   3,
						Usage:   "Number of models to export",
					},
				},
				Action: func(c *cli.Context) error {
					num := c.Int("num")
					fmt.Printf("📂 Exporting %d uncensored models...\n", num)
					err := cmd.Export(conf, num)
					if err != nil {
						slog.Error("Errore durante l'esportazione", "error", err)
						return err
					}
					fmt.Println("✅ Export completed successfully!")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Errore durante l'esecuzione dell'app", "error", err)
		os.Exit(1)
	}
}
