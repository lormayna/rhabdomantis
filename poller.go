package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)


type Response struct {
    Models []Model `json:"models"`
}

type Model struct {
    Name string `json:"name"`
}

type Host struct {
	IP     string
	Port   int
	Models []string
}

func ScanIP(host *Host) {
	url := fmt.Sprintf("http://%s:%d/api/tags", host.IP, host.Port)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Errore HTTP:", err)
		return
	}
	defer resp.Body.Close()

	var responseData Response
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		fmt.Println("Errore parsing JSON:", err)
		return
	}

	for _, model := range responseData.Models {
		host.Models = append(host.Models, model.Name)
	}
}