package protocol

import "encoding/json"

// Response is the standard response format from the TD server.
type Response struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Instance represents a running TouchDesigner instance discovered via heartbeat.
type Instance struct {
	ProjectPath string  `json:"projectPath"`
	ProjectName string  `json:"projectName"`
	Port        int     `json:"port"`
	PID         int     `json:"pid"`
	Timestamp   float64 `json:"timestamp"`
	TDVersion   string  `json:"tdVersion"`
	TDBuild     string  `json:"tdBuild"`
}

// HealthData is returned by the /health endpoint.
type HealthData struct {
	Version   string `json:"version"`
	Project   string `json:"project"`
	TDVersion string `json:"tdVersion"`
	TDBuild   string `json:"tdBuild"`
}
