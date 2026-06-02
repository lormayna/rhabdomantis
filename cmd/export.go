package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/lormayna/rhabdomantis/db"
	"github.com/lormayna/rhabdomantis/models"
)

// Pool di dati fittizi rigorosamente "uncensored" se il DB non ha abbastanza risultati
var (
	fakeUncensoredNames = []string{
		"wizard-vicuna-uncensored:30b",
		"dolphin-mixtral:8x7b-uncensored",
		"llama3-uncensored:8b",
		"mistral-uncensored:7b",
		"wizardlm-uncensored:13b",
		"nous-hermes-uncensored:13b",
	}
	fakeFamilies = map[string]string{
		"wizard-vicuna-uncensored:30b":    "llama",
		"dolphin-mixtral:8x7b-uncensored": "mixtral",
		"llama3-uncensored:8b":            "llama",
		"mistral-uncensored:7b":           "mistral",
		"wizardlm-uncensored:13b":         "llama",
		"nous-hermes-uncensored:13b":      "llama",
	}
	fakeSizes = map[string]int64{
		"wizard-vicuna-uncensored:30b":    19500667000,
		"dolphin-mixtral:8x7b-uncensored": 26000443000,
		"llama3-uncensored:8b":            4920551234,
		"mistral-uncensored:7b":           4100223000,
		"wizardlm-uncensored:13b":         7400331000,
		"nous-hermes-uncensored:13b":      7400552000,
	}
	fakeParamSizes = map[string]string{
		"wizard-vicuna-uncensored:30b":    "30B",
		"dolphin-mixtral:8x7b-uncensored": "8x7B",
		"llama3-uncensored:8b":            "8B",
		"mistral-uncensored:7b":           "7B",
		"wizardlm-uncensored:13b":         "13B",
		"nous-hermes-uncensored:13b":      "13B",
	}
)

func Export(conf *Config, numModels int) error {
	dbConn, err := sql.Open("sqlite3", conf.DBFile)
	if err != nil {
		return err
	}
	defer dbConn.Close()
	queries := db.New(dbConn)
	ctx := context.Background()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 1. Query con filtro LIKE per cercare solo i modelli con "uncensored" nel nome
	rows, err := queries.GetUncensoredModels(ctx)
	if err != nil {
		return err
	}

	var exportedModels []models.Model

	// Estrai i dati filtrati dal DB fino al limite richiesto
	for _, row := range rows {
		if len(exportedModels) >= numModels {
			break
		}

		name := row.Name
		size := row.Size.Int64
		family := row.Family.String
		paramSize := row.ParameterSize.String
		digest := row.Digest.String
		createdAt := row.CreatedAt

		if family == "" {
			family = "llama"
		}
		if paramSize == "" {
			paramSize = "8B"
		}
		if digest == "" {
			digest = generateFakeDigest(r)
		}

		quantLevel := "Q4_K_M"
		if strings.Contains(strings.ToLower(name), "q4_k_s") {
			quantLevel = "Q4_K_S"
		} else if strings.Contains(strings.ToLower(name), "q8") {
			quantLevel = "Q8_0"
		}

		exportedModels = append(exportedModels, models.Model{
			Name:       name,
			Model:      name,
			ModifiedAt: createdAt,
			Size:       size,
			Digest:     digest,
			Details: models.ModelDetails{
				Format:            "gguf",
				Family:            family,
				Families:          []string{family},
				ParameterSize:     paramSize,
				QuantizationLevel: quantLevel,
			},
		})
	}

	// 2. Se i risultati del DB non bastano, compensiamo con il pool "uncensored" fittizio
	if len(exportedModels) < numModels {
		needed := numModels - len(exportedModels)
		log.Printf("Trovati %d modelli corrispondenti nel DB. Generazione di %d modelli uncensored fittizi...", len(exportedModels), needed)

		for i := 0; i < needed; i++ {
			// Pesca un modello uncensored dal pool casuale
			baseModel := fakeUncensoredNames[r.Intn(len(fakeUncensoredNames))]

			randomDays := r.Intn(60)
			modTime := time.Now().AddDate(0, 0, -randomDays).Add(time.Duration(r.Intn(24)) * time.Hour)
			family := fakeFamilies[baseModel]

			fakeModel := models.Model{
				Name:       baseModel,
				Model:      baseModel,
				ModifiedAt: modTime,
				Size:       fakeSizes[baseModel] + int64(r.Intn(100000)),
				Digest:     generateFakeDigest(r),
				Details: models.ModelDetails{
					Format:            "gguf",
					Family:            family,
					Families:          []string{family},
					ParameterSize:     fakeParamSizes[baseModel],
					QuantizationLevel: "Q4_K_M",
				},
			}
			exportedModels = append(exportedModels, fakeModel)
		}
	}

	// 3. Generazione del JSON strutturato
	jsonBytes, err := json.MarshalIndent(exportedModels, "", "  ")
	if err != nil {
		return err
	}

	// 4. Scrittura del file models.json
	outputFile := "models.json"
	err = os.WriteFile(outputFile, jsonBytes, 0644)
	if err != nil {
		return err
	}

	log.Printf("Esportazione completata! File '%s' generato con %d modelli (tutti di tipo 'uncensored').", outputFile, len(exportedModels))
	return nil
}

func generateFakeDigest(r *rand.Rand) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, 64)
	for i := range b {
		b[i] = hexChars[r.Intn(len(hexChars))]
	}
	return string(b)
}
