package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/lormayna/rhabdomantis/db"
	"github.com/lormayna/rhabdomantis/models"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/sync/errgroup"
)

func Check(conf *Config) error {
	queries := db.New(conf.DBConn)
	hosts, err := queries.GetIPs(context.Background())
	if err != nil {
		slog.Error("Errore on retrieving data from DB", "error", err)
		return err
	}
	slog.Info("Hosts retrieved successfully", "count", len(hosts))
	g, _ := errgroup.WithContext(context.Background())

	// Definiamo il limite di concorrenza (es. 3 worker)
	g.SetLimit(conf.Workers)
	client := &http.Client{Timeout: 5 * time.Second}
	for _, host := range hosts {
		ip := host.Ip
		port := host.Port
		g.Go(func() error {
			hostPort := net.JoinHostPort(ip, fmt.Sprint(port))
			url := fmt.Sprintf("http://%s/api/tags", hostPort)
			resp, err := client.Get(url)
			if err != nil {
				slog.Error("Errore HTTP:", "error", err, "url", url)
				queries.UpdateHostInactive(context.Background(), ip)
				return nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Error("Non-200 response", "status", resp.StatusCode, "url", url)
				queries.UpdateHostActive(context.Background(), ip)
				return nil
			}
			slog.Info("Host is active", "ip", ip, "port", port)
			queries.UpdateHostActive(context.Background(), ip)

			var ollamaResp models.OllamaResponse
			if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
				slog.Error("Errore parsing", "ip", ip, "err", err)
				return nil
			}
			modelsJSON, _ := json.Marshal(ollamaResp.Models)
			slog.Info("Modelli trovati", "ip", ip, "models", string(modelsJSON))
			for _, m := range ollamaResp.Models {
				err = queries.SaveModel(context.Background(), db.SaveModelParams{
					Ip:            ip,
					Name:          m.Name,
					Size:          sql.NullInt64{Int64: m.Size, Valid: true},
					Family:        sql.NullString{String: m.Details.Family, Valid: true},
					ParameterSize: sql.NullString{String: m.Details.ParameterSize, Valid: true},
					Digest:        sql.NullString{String: m.Digest, Valid: true},
				})

				if err != nil {
					slog.Error("Errore salvataggio modello", "ip", ip, "model", m.Name, "err", err)
				}
			}

			return nil
		})
	}
	return g.Wait()
}
