package executors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// RunProxmox führt einen API-Call gegen einen Proxmox-Server aus
func RunProxmox(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer) int {
	host, _ := product["host"].(string)        // z.B. https://proxmox.example.com:8006
	node, _ := product["node"].(string)        // z.B. pve
	typeStr, _ := product["type"].(string)     // "qemu" oder "lxc"
	vmid, _ := product["vmid"].(string)        // z.B. 101 (optional für create)
	tokenID, _ := product["token_id"].(string) // z.B. root@pam!apitoken
	tokenSecret, _ := product["token_secret"].(string)
	apiCommand, _ := product["api_command"].(string) // z.B. "status/start", "create", "config", "agent/exec"
	apiParams, _ := product["api_params"].(map[string]interface{})

	if host == "" || node == "" || typeStr == "" || tokenID == "" || tokenSecret == "" || apiCommand == "" {
		io.WriteString(logWriter, "ERROR: Fehlende Proxmox-Parameter im Job\n")
		return 1
	}

	// Baue die API-URL
	var url string
	if apiCommand == "create" {
		url = fmt.Sprintf("%s/api2/json/nodes/%s/%s/%s", host, node, typeStr, apiCommand)
	} else if vmid != "" {
		url = fmt.Sprintf("%s/api2/json/nodes/%s/%s/%s/%s", host, node, typeStr, vmid, apiCommand)
	} else {
		url = fmt.Sprintf("%s/api2/json/nodes/%s/%s/%s", host, node, typeStr, apiCommand)
	}

	var reqBody io.Reader
	if apiParams != nil {
		jsonData, _ := json.Marshal(apiParams)
		reqBody = bytes.NewReader(jsonData)
	}
	method := "POST"
	if apiCommand == "status/current" || apiCommand == "config" { // Beispiel für GET
		method = "GET"
	}
	client := &http.Client{}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		io.WriteString(logWriter, "ERROR: Proxmox-Request-Fehler: "+err.Error()+"\n")
		return 1
	}
	req.Header.Set("Authorization", "PVEAPIToken="+tokenID+"="+tokenSecret)
	if apiParams != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		io.WriteString(logWriter, "ERROR: Proxmox-API-Fehler: "+err.Error()+"\n")
		return 1
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	io.WriteString(logWriter, fmt.Sprintf("Proxmox-API-Status: %s\n", resp.Status))
	io.WriteString(logWriter, string(body)+"\n")
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0
	}
	return 1
}
