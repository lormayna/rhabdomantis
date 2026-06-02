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

func Scan(conf *Config) error {
	dbConn, err := sql.Open("sqlite3", conf.DBFile)
	if err != nil {
		return err
	}
	defer dbConn.Close()
	queries := db.New(dbConn)
	ctx := context.Background()
	hosts, err := queries.GetIPs(ctx)
	if err != nil {
		slog.Error("Errore on retrieving data from DB", "error", err)
		return err
	}
	slog.Info("Hosts retrieved successfully", "count", len(hosts))
	g, ctx := errgroup.WithContext(ctx)

	// Definiamo il limite di concorrenza (es. 3 worker)
	g.SetLimit(conf.Workers)
	client := &http.Client{Timeout: 10 * time.Second}
	for _, host := range hosts {
		h := host
		g.Go(func() error {
			ip := h.Ip
			port := h.Port
			hostPort := net.JoinHostPort(ip, fmt.Sprint(port))
			url := fmt.Sprintf("http://%s/api/tags", hostPort)

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				slog.Error("Errore HTTP", "error", err, "url", url)
				queries.UpdateHostInactive(ctx, ip)
				return nil // Don't fail the whole group for one host
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Warn("Non-200 response", "status", resp.StatusCode, "url", url)
				queries.UpdateHostInactive(ctx, ip)
				return nil
			}

			slog.Info("Host is active", "ip", ip, "port", port)
			queries.UpdateHostActive(ctx, ip)

			var ollamaResp models.OllamaResponse
			if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
				slog.Error("Errore parsing JSON", "ip", ip, "error", err)
				return nil
			}

			for _, m := range ollamaResp.Models {
				err = queries.SaveModel(ctx, db.SaveModelParams{
					Ip:            ip,
					Name:          m.Name,
					Size:          sql.NullInt64{Int64: m.Size, Valid: true},
					Family:        sql.NullString{String: m.Details.Family, Valid: true},
					ParameterSize: sql.NullString{String: m.Details.ParameterSize, Valid: true},
					Digest:        sql.NullString{String: m.Digest, Valid: true},
				})

				if err != nil {
					slog.Error("Errore salvataggio modello", "ip", ip, "model", m.Name, "error", err)
				}
			}

			return nil
		})
	}
	return g.Wait()
}
