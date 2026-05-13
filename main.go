package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/lormayna/rhabdomantis/cmd"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

const workers = 3

func main() {
	shodan_api_key := "p5u2hw6AHFlXJLewlxwyZ8q9yygfwUMH"
	if shodan_api_key == "" {
		fmt.Println("Error: SHODAN_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	// Apri connessione al database SQLite
	dbConn, err := sql.Open("sqlite3", "hosts.db")
	if err != nil {
		slog.Error("Errore nell'apertura del database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	conf := &cmd.Config{
		ShodanAPIKey: shodan_api_key,
		DBConn:       dbConn,
		Workers:      workers,
	}

	// Crea la tabella se non esiste
	schema := `
    CREATE TABLE IF NOT EXISTS hosts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ip TEXT NOT NULL UNIQUE,
        port INTEGER NOT NULL,
        isp TEXT,
        asn TEXT,
        country TEXT,
        city TEXT,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        active BOOLEAN NOT NULL DEFAULT TRUE,
        scanned_at DATETIME
    );`
	_, err = dbConn.Exec(schema)
	if err != nil {
		slog.Error("Errore nella creazione della tabella", "error", err)
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

					err := cmd.Check(conf)
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
					if c.Bool("force") {
						fmt.Println("♻️  Sincronizzazione forzata in corso...")
					} else {
						fmt.Println("♻️  Sincronizzazione standard in corso...")
					}
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
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Errore durante l'esecuzione dell'app", "error", err)
		os.Exit(1)
	}
}
