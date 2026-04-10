package main

package main

import (
    "fmt"
    "io"
    "net/http"
)


type Response struct {
    Models []Model `json:"models"`
}

type Model struct {
    Name string `json:"name"`
}

struct Host struct {
	IP   string
	Port int
	Models []String
}

func ScanIP(host Host) {
	url := fmt.Sprintf("https://%s:%s/api/tags", host.IP, host.Port)
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("Errore:", err)
        return
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Errore lettura body:", err)
        return
    }

    var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		fmt.Println("Errore parsing JSON:", err)
		return
	}
	for _, model := range resp.Models {
		host.Models = append(host.Models, model.Name)
	}
}