package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	PathPattern   string        `yaml:"path_pattern"`
	MaxFiles      int           `yaml:"max_files"`
	ApiEndpoint   string        `yaml:"api_endpoint"`
	ApiToken      string        `yaml:"api_token"`
	ServerKey     string        `yaml:"server_key"`
	CheckInterval time.Duration `yaml:"check_interval"`
}

type Payload struct {
	Data []map[string]interface{} `json:"data"`
}

var config Config

func main() {
	configPath := flag.String("config", "/etc/airbox/shipper.yml", "Path to the configuration file")
	flag.Parse()

	loadConfig(*configPath)
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		processFiles()
	}
}

func loadConfig(configPath string) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		os.Exit(1)
	}
}

func processFiles() {
	files, err := filepath.Glob(config.PathPattern)
	if err != nil {
		fmt.Printf("Error reading files: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No files to process.")
		return
	}

	// Limit the number of files to process to maxFiles
	if len(files) > config.MaxFiles {
		files = files[:config.MaxFiles]
	}

	var payloadData []map[string]interface{}

	// Read each file and append its content to the payload
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(content, &data); err != nil {
			fmt.Printf("Error unmarshalling file %s: %v\n", file, err)
			continue
		}

		payloadData = append(payloadData, data)
	}

	if len(payloadData) == 0 {
		fmt.Println("No valid data to send.")
		return
	}

	payload := Payload{Data: payloadData}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling payload: %v\n", err)
		return
	}

	success := sendPayload(payloadBytes)
	if success {
		// Delete the processed files
		for _, file := range files {
			if err := os.Remove(file); err != nil {
				fmt.Printf("Error deleting file %s: %v\n", file, err)
			} else {
				fmt.Printf("Deleted file %s\n", file)
			}
		}
	}
}

func sendPayload(payload []byte) bool {
	client := &http.Client{}
	req, err := http.NewRequest("POST", config.ApiEndpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return false
	}

	req.Header.Set("Accept", "application/vnd.airbox.v1+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.ApiToken))
	req.Header.Set("X-Server-Key", config.ServerKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Received non-OK response: %d\n", resp.StatusCode)
		return false
	}

	fmt.Println("Payload successfully sent.")
	return true
}
