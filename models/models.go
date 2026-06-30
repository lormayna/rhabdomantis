package models

import (
	"time"
)

type OllamaResponse struct {
	Models []Model `json:"models"`
}

type Model struct {
	Name       string       `json:"name"`
	Model      string       `json:"model"`
	ModifiedAt time.Time    `json:"modified_at"`
	Size       int64        `json:"size"`
	Digest     string       `json:"digest"`
	Details    ModelDetails `json:"details"`
	// Campi opzionali (presenti solo in alcuni modelli nel tuo JSON)
	RemoteModel string `json:"remote_model,omitempty"`
	RemoteHost  string `json:"remote_host,omitempty"`
}

type ModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}
