package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lormayna/rhabdomantis/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ns3777k/go-shodan/v4/shodan"
)

func main() {
	//shodan_api_key := os.Getenv("SHODAN_API_KEY")
	shodan_api_key := "p5u2hw6AHFlXJLewlxwyZ8q9yygfwUMH"
	if shodan_api_key == "" {
		fmt.Println("Error: SHODAN_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	// Apri connessione al database SQLite
	dbConn, err := sql.Open("sqlite3", "hosts.db")
	if err != nil {
		log.Fatalf("Errore nell'apertura del database: %v", err)
	}
	defer dbConn.Close()

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
		log.Fatalf("Errore nella creazione della tabella: %v", err)
	}

	// Crea queries
	queries := db.New(dbConn)

	client := shodan.NewClient(nil, shodan_api_key)
	query := "product:Ollama"
	for i := range 100 {
		fmt.Printf("Pagina %d", i)
		hostQueryOptions := shodan.HostQueryOptions{Query: query, Page: i}
		result, err := client.GetHostsForQuery(context.Background(), &hostQueryOptions)
		if err != nil {
			log.Fatalf("Errore durante la ricerca: %v", err)
		}
		fmt.Printf("Pagina %d - Risultati totali trovati: %d\n", i, result.Total)
		// Iteriamo sui match trovati
		for _, host := range result.Matches {
			fmt.Printf("IP: %s | Porta: %d | Country: %s | City: %s | ASN: %s | ISP: %s\n",
				host.IP.String(),
				host.Port,
				host.Location.Country,
				host.Location.City,
				host.ASN,
				host.ISP,
			)

			// Inserisci nel database
			params := db.InsertHostParams{
				Ip:   host.IP.String(),
				Port: int64(host.Port),
				Isp: sql.NullString{
					String: host.ISP,
					Valid:  host.ISP != "",
				},
				Asn: sql.NullString{
					String: host.ASN,
					Valid:  host.ASN != "",
				},
				Country: sql.NullString{
					String: host.Location.Country,
					Valid:  host.Location.Country != "",
				},
				City: sql.NullString{
					String: host.Location.City,
					Valid:  host.Location.City != "",
				},
				ScannedAt: sql.NullTime{
					Time:  time.Now(),
					Valid: true,
				},
			}
			err = queries.InsertHost(context.Background(), params)
			if err != nil {
				log.Printf("Errore nell'inserimento dell'host %s: %v", host.IP.String(), err)
			}
		}
	}

}
