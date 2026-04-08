package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/alexflint/go-arg"
)

type CliArgs struct {
	Network  string `arg:"--network" help:"The network to scan"`
	Hostfile string `arg:"--hostfile" help:"The file to write the hosts to"`
}

func (args *CliArgs) Validate() error {
	if args.Network == "" && args.Hostfile == "" {
		return fmt.Errorf("devi specificare almeno --network o --hostfile")
	}
	return nil
}

func sendHTTPRequest(url string) (string, error) {
	resp, err := http.Get(fmt.Sprintf(url))
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Received non-OK response: %s", resp.Status)
	}
	return string(body), nil
}

func main() {
	var args CliArgs
	arg.MustParse(&args)

	fmt.Printf("network: %s\n", args.Network)
	fmt.Printf("hostfile: %s\n", args.Hostfile)
	err := args.Validate()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(99)
	}
	if args.Network != "" {
		fmt.Printf("Scanning network: %s\n", args.Network)

	}
}
