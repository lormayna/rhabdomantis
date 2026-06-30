package cmd

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/lormayna/rhabdomantis/db"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ns3777k/go-shodan/v4/shodan"
)

func Sync(conf *Config) error {
	//shodan_api_key := os.Getenv("SHODAN_API_KEY")

	shodan_api_key := conf.ShodanAPIKey
	if shodan_api_key == "" {
		return errors.New("Missing Shodan key")
	}
	dbConn, err := sql.Open("sqlite3", conf.DBFile)
	if err != nil {
		slog.Error("Errore nell'apertura del database", "error", err)
		return err
	}
	defer dbConn.Close()
	queries := db.New(dbConn)
	client := shodan.NewClient(nil, shodan_api_key)
	query := "product:Ollama"

	page := 1
	for {
		slog.Info("Fetching hosts from Shodan", "page", page)
		hostQueryOptions := shodan.HostQueryOptions{Query: query, Page: page}
		result, err := client.GetHostsForQuery(context.Background(), &hostQueryOptions)
		if err != nil {
			slog.Error("Errore durante la ricerca", "error", err, "page", page)
			return err
		}

		slog.Info("Risultati trovati", "page", page, "count", len(result.Matches), "total", result.Total)

		// Iteriamo sui match trovati
		for _, host := range result.Matches {
			slog.Debug("Host trovato",
				"ip", host.IP.String(),
				"port", host.Port,
				"country", host.Location.Country,
				"city", host.Location.City,
				"asn", host.ASN,
				"isp", host.ISP,
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
				var sqliteErr sqlite3.Error
				if errors.As(err, &sqliteErr) {
					if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique || sqliteErr.Code == sqlite3.ErrConstraint {
						slog.Debug("Host already exists", "ip", host.IP.String())
						continue
					}
				}
				slog.Error("Errore nell'inserimento dell'host", "ip", host.IP.String(), "error", err)
			}
		}

		// Verifichiamo se ci sono altre pagine
		if len(result.Matches) == 0 || (page*100) >= result.Total {
			break
		}
		page++
	}
	return nil
}
