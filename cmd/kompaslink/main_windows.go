package main

import (
	"encoding/json"
	"fmt"
	"os"

	"kompaslink/internal/kompas"
)

func main() {
	os.Exit(run())
}

func run() int {
	report, probeErr := kompas.Probe()
	output := struct {
		Report any    `json:"report"`
		Error  string `json:"error,omitempty"`
	}{Report: report}
	if probeErr != nil {
		output.Error = probeErr.Error()
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, "Не удалось вывести диагностику:", err)
		return 1
	}
	if probeErr != nil {
		return 1
	}
	return 0
}
