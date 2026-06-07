package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/lormayna/rhabdomantis/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ollama/ollama/api"
	"golang.org/x/sync/errgroup"
)

//go:embed prompts/verify.tmpl
var fs embed.FS

const sumTemplate = "prompts/verify.tmpl"

func RenderPrompt(t *template.Template, a, b int) (string, error) {
	var buf bytes.Buffer
	err := t.Execute(&buf, struct {
		A int
		B int
	}{
		A: a,
		B: b,
	})
	return buf.String(), err
}

func Verify(conf *Config) error {
	promptTmpl := template.Must(template.ParseFS(fs, sumTemplate))
	dbConn, err := sql.Open("sqlite3", conf.DBFile)
	if err != nil {
		return err
	}
	defer dbConn.Close()
	queries := db.New(dbConn)
	hosts, err := queries.GetIPs(context.Background())
	if err != nil {
		slog.Error("Errore nel recupero dati dal DB", "error", err)
		return err
	}
	slog.Info("Host attivi recuperati con successo", "count", len(hosts))

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(conf.Workers)

	for _, host := range hosts {

		g.Go(func() error {
			h := host
			model, err := queries.GetRandomModelByIP(ctx, h.Ip)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				slog.Warn("Nessun modello trovato per l'host", "ip", h.Ip)
				slog.Error("Errore nel recupero del modello dal DB", "error", err)
				return nil
			}
			slog.Info("Modello recuperato", "ip", h.Ip, "model", model.Name)

			hostPort := net.JoinHostPort(h.Ip, fmt.Sprint(h.Port))

			// SSL Detection
			sslEnabled := false
			finalURLStr := fmt.Sprintf("http://%s", hostPort)

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			httpClient := &http.Client{Transport: tr, Timeout: 10 * time.Second}

			resp, err := httpClient.Get(fmt.Sprintf("https://%s/api/tags", hostPort))
			if err == nil {
				resp.Body.Close()
				sslEnabled = true
				finalURLStr = fmt.Sprintf("https://%s", hostPort)
				slog.Info("SSL rilevato", "ip", h.Ip)
			}

			err = queries.UpdateHostSSL(ctx, db.UpdateHostSSLParams{
				SslEnabled: sslEnabled,
				Ip:         h.Ip,
			})
			if err != nil {
				slog.Error("Errore aggiornamento SSL nel DB", "error", err, "ip", h.Ip)
			}

			remoteURL, err := url.Parse(finalURLStr)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				slog.Error("Errore nella costruzione dell'URL", "error", err, "ip", h.Ip)
				return nil
			}

			client := api.NewClient(remoteURL, httpClient)

			nA, _ := rand.Int(rand.Reader, big.NewInt(100))
			nB, _ := rand.Int(rand.Reader, big.NewInt(100))
			a, b := int(nA.Int64()), int(nB.Int64())
			expectedSum := a + b

			content, err := RenderPrompt(promptTmpl, a, b)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				slog.Error("Errore nella generazione del prompt", "error", err, "ip", h.Ip)
				return nil
			}

			// FIX: Corretta la definizione della richiesta
			stream := false
			req := &api.ChatRequest{
				Model: model.Name,
				Messages: []api.Message{
					{Role: "user", Content: content},
				},
				Stream: &stream,
			}

			err = client.Chat(ctx, req, func(resp api.ChatResponse) error {
				reply := resp.Message.Content
				cleanReply := strings.TrimSpace(reply)

				var verdict string
				sumValue, err := strconv.Atoi(cleanReply)
				if err != nil {
					slog.Warn("Risposta non numerica", "reply", cleanReply, "ip", h.Ip)
					verdict = "pending"
				} else if sumValue == expectedSum {
					verdict = "success"
				} else {
					verdict = "failed"
				}

				// FIX: Corretti virgole e nomi campi struct
				params := db.SaveInferenceParams{
					ModelID:          model.ID, // Assicurati che sia Name se viene da sqlc
					Prompt:           content,
					Response:         sql.NullString{Valid: true, String: reply},
					TotalDurationMs:  sql.NullInt64{Valid: true, Int64: resp.TotalDuration.Milliseconds()},
					PromptTokens:     sql.NullInt64{Valid: true, Int64: int64(resp.PromptEvalCount)},
					CompletionTokens: sql.NullInt64{Valid: true, Int64: int64(resp.EvalCount)},
					Verdict:          sql.NullString{Valid: true, String: verdict},
					HttpStatusCode:   sql.NullInt64{Valid: true, Int64: int64(200)},
					Notes:            sql.NullString{Valid: true, String: ""},
				}

				err = queries.SaveInference(ctx, params)

				if err != nil {
					slog.Error("Errore salvataggio DB", "error", err)
				}

				return nil
			})
			if err != nil {
				var notes string
				var statusCode int64

				if apiErr, ok := err.(*api.StatusError); ok {
					notes = apiErr.Error()
					statusCode = int64(apiErr.StatusCode)
				}

				params := db.SaveInferenceParams{
					ModelID:          model.ID,
					Prompt:           content,
					Response:         sql.NullString{Valid: true, String: notes},
					TotalDurationMs:  sql.NullInt64{Valid: false, Int64: 0},
					PromptTokens:     sql.NullInt64{Valid: true, Int64: int64(0)},
					CompletionTokens: sql.NullInt64{Valid: true, Int64: int64(0)},
					Verdict:          sql.NullString{Valid: true, String: "failed"},
					HttpStatusCode:   sql.NullInt64{Valid: true, Int64: statusCode},
					Notes:            sql.NullString{Valid: true, String: notes},
				}

				err = queries.SaveInference(ctx, params)

				if err != nil {
					slog.Error("Errore salvataggio DB", "error", err)
				}
				return nil
			}
			return nil
		})
	}

	return g.Wait()
}
