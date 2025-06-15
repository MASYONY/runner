package executors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/MASYONY/runner/utils"
)

// RunProxmox mit Interpolation
func RunProxmox(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer, workDir string, jobResults map[string]map[string]interface{}, previousJobID string, jobIDMap map[string]string) int {
	workDir = utils.InterpolateVars(workDir, workDir, jobResults, previousJobID, jobIDMap, nil)
	if wd, ok := variables["WORKDIR"]; ok && wd != "" {
		workDir = utils.InterpolateVars(wd, workDir, jobResults, previousJobID, jobIDMap, nil)
	}

	host, _ := product["host"].(string)
	host = utils.InterpolateVars(host, workDir, jobResults, previousJobID, jobIDMap, nil)
	node, _ := product["node"].(string)
	node = utils.InterpolateVars(node, workDir, jobResults, previousJobID, jobIDMap, nil)
	typeStr, _ := product["type"].(string)
	typeStr = utils.InterpolateVars(typeStr, workDir, jobResults, previousJobID, jobIDMap, nil)
	vmid, _ := product["vmid"].(string)
	vmid = utils.InterpolateVars(vmid, workDir, jobResults, previousJobID, jobIDMap, nil)
	tokenID, _ := product["token_id"].(string)
	tokenID = utils.InterpolateVars(tokenID, workDir, jobResults, previousJobID, jobIDMap, nil)
	tokenSecret, _ := product["token_secret"].(string)
	tokenSecret = utils.InterpolateVars(tokenSecret, workDir, jobResults, previousJobID, jobIDMap, nil)
	apiCommand, _ := product["api_command"].(string)
	apiCommand = utils.InterpolateVars(apiCommand, workDir, jobResults, previousJobID, jobIDMap, nil)
	apiParams, _ := product["api_params"].(map[string]interface{})
	// Optional: Rekursive Interpolation für alle Strings in apiParams
	if apiParams != nil {
		for k, v := range apiParams {
			if str, ok := v.(string); ok {
				apiParams[k] = utils.InterpolateVars(str, workDir, jobResults, previousJobID, jobIDMap, nil)
			}
		}
	}

	// Interpolation für alle Variablenwerte (rekursiv, falls Platzhalter enthalten)
	for k, v := range variables {
		variables[k] = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
	}

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
	result := map[string]interface{}{
		"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
		"data":    string(body),
		"error":   "",
	}
	if !result["success"].(bool) {
		result["error"] = fmt.Sprintf("Proxmox-API-Status: %s", resp.Status)
	}
	_ = utils.WriteJobResult(jobID, workDir, result)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0
	}
	return 1
}
