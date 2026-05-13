package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lormayna/rhabdomantis/db"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ns3777k/go-shodan/v4/shodan"
)

func Scan(conf *Config) error {
	//shodan_api_key := os.Getenv("SHODAN_API_KEY")

	shodan_api_key := conf.ShodanAPIKey
	if shodan_api_key == "" {
		return errors.New("Missing Shodan key")
	}
	queries := db.New(conf.DBConn)
	client := shodan.NewClient(nil, shodan_api_key)
	query := "product:Ollama"
	for i := range 100 {
		fmt.Printf("Getting %d items", i)
		hostQueryOptions := shodan.HostQueryOptions{Query: query, Page: i}
		result, err := client.GetHostsForQuery(context.Background(), &hostQueryOptions)
		if err != nil {
			slog.Error("Errore durante la ricerca: %v", err)
			return err
		}
		slog.Info("Pagina %d - Risultati totali trovati: %d\n", i, result.Total)
		// Iteriamo sui match trovati
		for _, host := range result.Matches {
			slog.Info("IP: %s | Porta: %d | Country: %s | City: %s | ASN: %s | ISP: %s\n",
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
				if errors.Is(err, sqlite3.ErrConstraintUnique) {
					slog.Info("Host " + host.IP.String() + " already exists in the database, skipping")
					continue
				}
				slog.Error("Errore nell'inserimento dell'host %s: %v", host.IP.String(), err)
				return err
			}
		}
	}
	return nil
}
