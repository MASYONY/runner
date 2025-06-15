package executors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/MASYONY/runner/utils"
)

// RunSevDesk führt Aktionen gegen die sevDesk-API aus (z.B. Rechnung erstellen, stornieren, PDF, Versand, Kontakt)
func RunSevDesk(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer, workDir string, jobResults map[string]map[string]interface{}, previousJobID string, jobIDMap map[string]string) int {
	// Interpolation für alle String-Felder in product und variables
	for k, v := range product {
		if str, ok := v.(string); ok {
			product[k] = utils.InterpolateVars(str, workDir, jobResults, previousJobID, jobIDMap, nil)
		}
	}
	// Interpolation für alle Variablenwerte (rekursiv, falls Platzhalter enthalten)
	for k, v := range variables {
		variables[k] = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
	}

	apiToken, _ := product["api_token"].(string)
	// workDir := "./workdir"
	// if wd, ok := variables["WORKDIR"]; ok && wd != "" {
	// 	workDir = wd
	// }
	// io.WriteString(logWriter, fmt.Sprintf("sevDesk: Workdir: %s\n", workDir))

	client := &http.Client{}
	baseURL := "https://my.sevdesk.de/api/v1"

	switch product["type"] {
	case "create_invoice":
		contactID, _ := product["contact_id"].(string)
		invoiceData, _ := product["invoice_data"].(map[string]interface{})
		// Automatische Kontakterstellung, falls contact_id fehlt, aber contact_data vorhanden
		if contactID == "" {
			if contactData, ok := product["contact_data"].(map[string]interface{}); ok && contactData != nil {
				jsonContact, _ := json.Marshal(contactData)
				req, err := http.NewRequest("POST", baseURL+"/Contact", bytes.NewBuffer(jsonContact))
				if err != nil {
					io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Kontakt-Anfrage: "+err.Error()+"\n")
					return 1
				}
				req.Header.Set("Authorization", apiToken)
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					io.WriteString(logWriter, "sevDesk: API-Fehler bei Kontakt: "+err.Error()+"\n")
					return 1
				}
				defer resp.Body.Close()
				body, _ := ioutil.ReadAll(resp.Body)
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var result map[string]interface{}
					_ = json.Unmarshal(body, &result)
					if obj, ok := result["objects"].(map[string]interface{}); ok {
						if id, ok := obj["id"].(string); ok {
							contactID = id
							io.WriteString(logWriter, "sevDesk: Kontakt angelegt, ID: "+contactID+"\n")
						}
					}
				}
				if contactID == "" {
					io.WriteString(logWriter, "sevDesk: Konnte Kontakt nicht anlegen!\n")
					return 1
				}
			} else {
				io.WriteString(logWriter, "sevDesk: contact_id oder contact_data fehlt!\n")
				return 1
			}
		}
		if invoiceData == nil {
			io.WriteString(logWriter, "sevDesk: invoice_data fehlt!\n")
			return 1
		}
		payload := map[string]interface{}{
			"contact": map[string]interface{}{"id": contactID},
			"invoice": invoiceData,
		}
		jsonData, _ := json.Marshal(payload)
		req, err := http.NewRequest("POST", baseURL+"/Invoice", bytes.NewBuffer(jsonData))
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		// Debug: result.json nach dem Schreiben ausgeben
		resultPath := filepath.Join(workDir, jobID, "result.json")
		if resBytes, err := ioutil.ReadFile(resultPath); err == nil {
			io.WriteString(logWriter, "sevDesk: DEBUG result.json (nach WriteJobResult): "+string(resBytes)+"\n")
		} else {
			io.WriteString(logWriter, "sevDesk: DEBUG Fehler beim Lesen von result.json: "+err.Error()+"\n")
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "cancel_invoice":
		invoiceID, _ := product["invoice_id"].(string)
		if invoiceID == "" {
			io.WriteString(logWriter, "sevDesk: invoice_id fehlt!\n")
			return 1
		}
		payload := map[string]interface{}{"status": 100}
		jsonData, _ := json.Marshal(payload)
		req, err := http.NewRequest("PATCH", baseURL+"/Invoice/"+invoiceID, bytes.NewBuffer(jsonData))
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "get_invoice_pdf":
		invoiceID, _ := product["invoice_id"].(string)
		if invoiceID == "" {
			io.WriteString(logWriter, "sevDesk: invoice_id fehlt!\n")
			return 1
		}
		url := fmt.Sprintf("%s/Invoice/%s/getPdf", baseURL, invoiceID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			pdfBytes, _ := ioutil.ReadAll(resp.Body)
			io.WriteString(logWriter, fmt.Sprintf("sevDesk: PDF (%d bytes) geladen.\n", len(pdfBytes)))
			// Optional: PDF speichern
			if out, ok := product["pdf_output"].(string); ok && out != "" {
				ioutil.WriteFile(out, pdfBytes, 0644)
				io.WriteString(logWriter, "sevDesk: PDF gespeichert unter "+out+"\n")
			}
			return 0
		}
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.Copy(logWriter, resp.Body)
		return 1
	case "send_invoice":
		invoiceID, _ := product["invoice_id"].(string)
		if invoiceID == "" {
			io.WriteString(logWriter, "sevDesk: invoice_id fehlt!\n")
			return 1
		}
		url := fmt.Sprintf("%s/Invoice/%s/sendViaEmail", baseURL, invoiceID)
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "save_invoice_draft":
		invoiceData, _ := product["invoice_data"].(map[string]interface{})
		if invoiceData == nil {
			io.WriteString(logWriter, "sevDesk: invoice_data fehlt!\n")
			return 1
		}
		jsonData, _ := json.Marshal(invoiceData)
		req, err := http.NewRequest("POST", baseURL+"/Invoice", bytes.NewBuffer(jsonData))
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "create_contact":
		contactData, _ := product["contact_data"].(map[string]interface{})
		if contactData == nil {
			io.WriteString(logWriter, "sevDesk: contact_data fehlt!\n")
			return 1
		}
		jsonData, _ := json.Marshal(contactData)
		req, err := http.NewRequest("POST", baseURL+"/Contact", bytes.NewBuffer(jsonData))
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "get_invoice":
		invoiceID, _ := product["invoice_id"].(string)
		if invoiceID == "" {
			io.WriteString(logWriter, "sevDesk: invoice_id fehlt!\n")
			return 1
		}
		url := fmt.Sprintf("%s/Invoice/%s", baseURL, invoiceID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "delete_invoice":
		invoiceID, _ := product["invoice_id"].(string)
		if invoiceID == "" {
			io.WriteString(logWriter, "sevDesk: invoice_id fehlt!\n")
			return 1
		}
		url := fmt.Sprintf("%s/Invoice/%s", baseURL, invoiceID)
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "get_invoice_status":
		invoiceID, _ := product["invoice_id"].(string)
		if invoiceID == "" {
			io.WriteString(logWriter, "sevDesk: invoice_id fehlt!\n")
			return 1
		}
		url := fmt.Sprintf("%s/Invoice/%s", baseURL, invoiceID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	case "list_invoices":
		url := baseURL + "/Invoice"
		// Filter-Parameter als Query-String anhängen
		if filter, ok := product["filter"].(map[string]interface{}); ok && filter != nil {
			params := "?"
			for k, v := range filter {
				params += fmt.Sprintf("%s=%v&", k, v)
			}
			if len(params) > 1 {
				url += params[:len(params)-1] // letztes & entfernen
			}
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: Fehler beim Erstellen der Anfrage: "+err.Error()+"\n")
			return 1
		}
		req.Header.Set("Authorization", apiToken)
		resp, err := client.Do(req)
		if err != nil {
			io.WriteString(logWriter, "sevDesk: API-Fehler: "+err.Error()+"\n")
			return 1
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		io.WriteString(logWriter, fmt.Sprintf("sevDesk: Status %s\n", resp.Status))
		io.WriteString(logWriter, string(body)+"\n")
		result := map[string]interface{}{
			"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			"data":    string(body),
			"error":   "",
		}
		PatchSevDeskDataField(result, logWriter)
		if !result["success"].(bool) {
			result["error"] = fmt.Sprintf("sevDesk: Status %s", resp.Status)
		}
		_ = utils.WriteJobResult(jobID, workDir, result)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		}
		return 1
	default:
		io.WriteString(logWriter, "sevDesk: Unbekannter Typ!\n")
		return 1
	}
}

// PatchSevDeskDataField prüft, ob das Feld "data" im result-Objekt ein JSON-String ist.
// Falls ja, wird es dynamisch geparst und als Objekt/Array ersetzt.
// Funktioniert für alle möglichen Typen (string, Objekt, Array, null).
func PatchSevDeskDataField(result map[string]interface{}, logWriter io.Writer) {
	data, ok := result["data"]
	if !ok || data == nil {
		io.WriteString(logWriter, "sevDesk: DEBUG PatchSevDeskDataField: Kein data-Feld vorhanden oder nil\n")
		return
	}

	// Wenn data bereits ein Objekt oder Array ist, nichts tun
	switch v := data.(type) {
	case map[string]interface{}, []interface{}:
		io.WriteString(logWriter, "sevDesk: DEBUG PatchSevDeskDataField: data ist bereits Objekt/Array\n")
		return
	case string:
		// Prüfen, ob der String wie JSON aussieht
		trimmed := strings.TrimSpace(v)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			var parsed interface{}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				result["data"] = parsed
				io.WriteString(logWriter, "sevDesk: DEBUG PatchSevDeskDataField: data als JSON geparst\n")
				return
			} else {
				io.WriteString(logWriter, "sevDesk: DEBUG PatchSevDeskDataField: Fehler beim Parsen von data als JSON: "+err.Error()+"\n")
			}
		} else {
			io.WriteString(logWriter, "sevDesk: DEBUG PatchSevDeskDataField: data ist String, aber kein JSON\n")
		}
	default:
		io.WriteString(logWriter, "sevDesk: DEBUG PatchSevDeskDataField: data ist Typ "+fmt.Sprintf("%T", v)+"\n")
	}
}
