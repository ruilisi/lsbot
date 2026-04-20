package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Home Assistant tool implementation.
// Auth: HASS_TOKEN env var (Long-Lived Access Token).
// URL:  HASS_URL env var (default: http://homeassistant.local:8123).

var (
	haEntityIDRe    = regexp.MustCompile(`^[a-z_][a-z0-9_]*\.[a-z0-9_]+$`)
	haServiceNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	// Domains that allow arbitrary shell/code execution — always blocked.
	haBlockedDomains = map[string]bool{
		"shell_command": true,
		"command_line":  true,
		"python_script": true,
		"pyscript":      true,
		"hassio":        true,
		"rest_command":  true,
	}
)

func haConfig() (hassURL, token string) {
	hassURL = strings.TrimRight(os.Getenv("HASS_URL"), "/")
	if hassURL == "" {
		hassURL = "http://homeassistant.local:8123"
	}
	token = os.Getenv("HASS_TOKEN")
	return
}

func haRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	hassURL, token := haConfig()
	if token == "" {
		return nil, fmt.Errorf("HASS_TOKEN not set")
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, hassURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HA API %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// handleHAListEntities handles the ha_list_entities tool.
func handleHAListEntities(ctx context.Context, args map[string]any) string {
	domain, _ := args["domain"].(string)
	area, _ := args["area"].(string)

	data, err := haRequest(ctx, http.MethodGet, "/api/states", nil)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	var states []map[string]any
	if err := json.Unmarshal(data, &states); err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	var entities []map[string]any
	for _, s := range states {
		entityID, _ := s["entity_id"].(string)
		if domain != "" && !strings.HasPrefix(entityID, domain+".") {
			continue
		}
		attrs, _ := s["attributes"].(map[string]any)
		friendlyName, _ := attrs["friendly_name"].(string)
		if area != "" {
			areaLower := strings.ToLower(area)
			if !strings.Contains(strings.ToLower(friendlyName), areaLower) {
				areaVal, _ := attrs["area"].(string)
				if !strings.Contains(strings.ToLower(areaVal), areaLower) {
					continue
				}
			}
		}
		entities = append(entities, map[string]any{
			"entity_id":    entityID,
			"state":        s["state"],
			"friendly_name": friendlyName,
		})
	}

	out, _ := json.Marshal(map[string]any{
		"count":    len(entities),
		"entities": entities,
	})
	return string(out)
}

// handleHAGetState handles the ha_get_state tool.
func handleHAGetState(ctx context.Context, args map[string]any) string {
	entityID, _ := args["entity_id"].(string)
	if entityID == "" {
		return `{"error":"entity_id is required"}`
	}
	if !haEntityIDRe.MatchString(entityID) {
		return fmt.Sprintf(`{"error":"invalid entity_id format: %s"}`, entityID)
	}

	data, err := haRequest(ctx, http.MethodGet, "/api/states/"+entityID, nil)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	out, _ := json.Marshal(map[string]any{
		"entity_id":    state["entity_id"],
		"state":        state["state"],
		"attributes":   state["attributes"],
		"last_changed": state["last_changed"],
		"last_updated": state["last_updated"],
	})
	return string(out)
}

// handleHAListServices handles the ha_list_services tool.
func handleHAListServices(ctx context.Context, args map[string]any) string {
	filterDomain, _ := args["domain"].(string)

	data, err := haRequest(ctx, http.MethodGet, "/api/services", nil)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	var services []map[string]any
	if err := json.Unmarshal(data, &services); err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	var result []map[string]any
	for _, svc := range services {
		domain, _ := svc["domain"].(string)
		if filterDomain != "" && domain != filterDomain {
			continue
		}
		if haBlockedDomains[domain] {
			continue
		}
		result = append(result, svc)
	}

	out, _ := json.Marshal(map[string]any{
		"count":    len(result),
		"services": result,
	})
	return string(out)
}

// handleHACallService handles the ha_call_service tool.
func handleHACallService(ctx context.Context, args map[string]any) string {
	domain, _ := args["domain"].(string)
	service, _ := args["service"].(string)
	entityID, _ := args["entity_id"].(string)

	if domain == "" || service == "" {
		return `{"error":"domain and service are required"}`
	}
	if !haServiceNameRe.MatchString(domain) || !haServiceNameRe.MatchString(service) {
		return `{"error":"invalid domain or service name format"}`
	}
	if haBlockedDomains[domain] {
		return fmt.Sprintf(`{"error":"domain %q is blocked for security"}`, domain)
	}
	if entityID != "" && !haEntityIDRe.MatchString(entityID) {
		return fmt.Sprintf(`{"error":"invalid entity_id format: %s"}`, entityID)
	}

	payload := map[string]any{}
	if entityID != "" {
		payload["entity_id"] = entityID
	}
	if dataRaw, ok := args["data"]; ok && dataRaw != nil {
		if dataMap, ok := dataRaw.(map[string]any); ok {
			for k, v := range dataMap {
				payload[k] = v
			}
		}
	}

	data, err := haRequest(ctx, http.MethodPost,
		fmt.Sprintf("/api/services/%s/%s", domain, service), payload)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}

	var affected []map[string]any
	_ = json.Unmarshal(data, &affected)

	var summary []map[string]any
	for _, s := range affected {
		summary = append(summary, map[string]any{
			"entity_id": s["entity_id"],
			"state":     s["state"],
		})
	}
	out, _ := json.Marshal(map[string]any{
		"success":           true,
		"service":           domain + "." + service,
		"affected_entities": summary,
	})
	return string(out)
}
