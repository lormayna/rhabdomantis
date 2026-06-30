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

type ModelTarget struct {
	ID   int64
	Name string
}

func selectMostLikelyUncensored(models []ModelTarget) ModelTarget {
	if len(models) == 0 {
		return ModelTarget{}
	}

	bestModel := models[0]
	bestScore := -1

	for _, m := range models {
		score := 0
		nameLower := strings.ToLower(m.Name)

		if strings.Contains(nameLower, "uncensored") {
			score += 100
		}
		if strings.Contains(nameLower, "dolphin") {
			score += 50
		}
		if strings.Contains(nameLower, "hermes") {
			score += 40
		}
		if strings.Contains(nameLower, "wizard") {
			score += 30
		}
		if strings.Contains(nameLower, "vicuna") {
			score += 30
		}
		if strings.Contains(nameLower, "nous") {
			score += 20
		}

		if score > bestScore {
			bestScore = score
			bestModel = m
		}
	}

	return bestModel
}

func Verify(conf *Config, customPrompt string) error {
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
			var targets []ModelTarget

			if customPrompt != "" {
				modelsList, err := queries.GetModelsByHostIP(ctx, h.Ip)
				if err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					slog.Error("Errore nel recupero dei modelli per l'host dal DB", "error", err, "ip", h.Ip)
					return nil
				}
				if len(modelsList) == 0 {
					slog.Warn("Nessun modello trovato per l'host", "ip", h.Ip)
					return nil
				}
				var hostTargets []ModelTarget
				for _, m := range modelsList {
					hostTargets = append(hostTargets, ModelTarget{ID: m.ID, Name: m.Name})
				}
				selected := selectMostLikelyUncensored(hostTargets)
				targets = append(targets, selected)
			} else {
				model, err := queries.GetRandomModelByIP(ctx, h.Ip)
				if err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					slog.Warn("Nessun modello trovato per l'host", "ip", h.Ip)
					slog.Error("Errore nel recupero del modello dal DB", "error", err)
					return nil
				}
				targets = append(targets, ModelTarget{ID: model.ID, Name: model.Name})
			}

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

			for _, target := range targets {
				if customPrompt != "" {
					stream := false
					req := &api.ChatRequest{
						Model: target.Name,
						Messages: []api.Message{
							{Role: "user", Content: customPrompt},
						},
						Stream: &stream,
					}
					slog.Info("Invio prompt personalizzato", "ip", h.Ip, "model", target.Name, "prompt", customPrompt)
					err = client.Chat(ctx, req, func(resp api.ChatResponse) error {
						reply := resp.Message.Content
						params := db.SaveCustomInferenceParams{
							Ip:      h.Ip,
							ModelID: target.ID,
							Prompt:  customPrompt,
							Reply:   sql.NullString{Valid: true, String: reply},
						}
						err = queries.SaveCustomInference(ctx, params)
						if err != nil {
							slog.Error("Errore salvataggio prompt personalizzato nel DB", "error", err)
						}
						return nil
					})
					if err != nil {
						var notes string
						if apiErr, ok := err.(*api.StatusError); ok {
							notes = apiErr.Error()
						} else {
							notes = err.Error()
						}
						slog.Error("Errore chat prompt personalizzato", "error", err, "ip", h.Ip, "model", target.Name)
						params := db.SaveCustomInferenceParams{
							Ip:      h.Ip,
							ModelID: target.ID,
							Prompt:  customPrompt,
							Reply:   sql.NullString{Valid: true, String: "Error: " + notes},
						}
						_ = queries.SaveCustomInference(ctx, params)
					}
				} else {
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
						continue
					}

					stream := false
					req := &api.ChatRequest{
						Model: target.Name,
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

						params := db.SaveInferenceParams{
							ModelID:          target.ID,
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
							ModelID:          target.ID,
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
					}
				}
			}
			return nil
		})
	}

	return g.Wait()
}
