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
	ProjectPath          string  `json:"projectPath"`
	ProjectName          string  `json:"projectName"`
	Port                 int     `json:"port"`
	PID                  int     `json:"pid"`
	Timestamp            float64 `json:"timestamp"`
	TDVersion            string  `json:"tdVersion"`
	TDBuild              string  `json:"tdBuild"`
	State                string  `json:"state"` // ready, cooking, error, initializing, playing, paused
	ConnectorName        string  `json:"connectorName"`
	ConnectorVersion     string  `json:"connectorVersion"`
	ProtocolVersion      int     `json:"protocolVersion"`
	ConnectorInstallMode string  `json:"connectorInstallMode"`
}

// HealthData is returned by the /health endpoint.
type HealthData struct {
	Version              string `json:"version"`
	Project              string `json:"project"`
	TDVersion            string `json:"tdVersion"`
	TDBuild              string `json:"tdBuild"`
	ConnectorName        string `json:"connectorName"`
	ConnectorVersion     string `json:"connectorVersion"`
	ProtocolVersion      int    `json:"protocolVersion"`
	ConnectorInstallMode string `json:"connectorInstallMode"`
}

// HarnessCapabilitiesRequest requests the active harness capabilities surface.
type HarnessCapabilitiesRequest struct {
	Scope     string   `json:"scope,omitempty"`
	Goal      string   `json:"goal,omitempty"`
	Include   []string `json:"include,omitempty"`
	RequestID string   `json:"requestId,omitempty"`
}

// HarnessObserveRequest requests an agent-oriented observation snapshot.
type HarnessObserveRequest struct {
	Scope     string   `json:"scope,omitempty"`
	Goal      string   `json:"goal,omitempty"`
	Depth     int      `json:"depth,omitempty"`
	Include   []string `json:"include,omitempty"`
	RequestID string   `json:"requestId,omitempty"`
}

// HarnessCheck describes a verification assertion for the harness.
type HarnessCheck struct {
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Target   string `json:"target,omitempty"`
	Path     string `json:"path,omitempty"`
	Param    string `json:"param,omitempty"`
	Expected string `json:"expected,omitempty"`
}

// HarnessVerifyRequest requests harness verification over a scope.
type HarnessVerifyRequest struct {
	Scope     string         `json:"scope,omitempty"`
	Goal      string         `json:"goal,omitempty"`
	Checks    []HarnessCheck `json:"checks,omitempty"`
	RequestID string         `json:"requestId,omitempty"`
}

// HarnessOperation describes a planned or applied harness operation.
type HarnessOperation struct {
	Action  string                 `json:"action,omitempty"`
	Target  string                 `json:"target,omitempty"`
	Path    string                 `json:"path,omitempty"`
	Family  string                 `json:"family,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Summary string                 `json:"summary,omitempty"`
}

// HarnessApplyRequest requests a patch/apply step.
type HarnessApplyRequest struct {
	Scope      string             `json:"scope,omitempty"`
	Goal       string             `json:"goal,omitempty"`
	DryRun     bool               `json:"dryRun,omitempty"`
	Operations []HarnessOperation `json:"operations,omitempty"`
	RequestID  string             `json:"requestId,omitempty"`
}

// HarnessRollbackRequest requests rollback of a prior harness change.
type HarnessRollbackRequest struct {
	Handle    string `json:"handle,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// HarnessHistoryRequest requests recent harness activity.
type HarnessHistoryRequest struct {
	Scope     string `json:"scope,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// HarnessTarget identifies an output, viewer, or other important node.
type HarnessTarget struct {
	Label  string `json:"label,omitempty"`
	Path   string `json:"path,omitempty"`
	Type   string `json:"type,omitempty"`
	Family string `json:"family,omitempty"`
	Role   string `json:"role,omitempty"`
	Active bool   `json:"active,omitempty"`
}

// HarnessMetric captures a summarized metric or hotspot.
type HarnessMetric struct {
	Name     string  `json:"name,omitempty"`
	Value    float64 `json:"value,omitempty"`
	Unit     string  `json:"unit,omitempty"`
	Severity string  `json:"severity,omitempty"`
	Target   string  `json:"target,omitempty"`
	Summary  string  `json:"summary,omitempty"`
}

// HarnessChange summarizes a single harness change.
type HarnessChange struct {
	Action  string `json:"action,omitempty"`
	Target  string `json:"target,omitempty"`
	Path    string `json:"path,omitempty"`
	Family  string `json:"family,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// HarnessCheckpoint points to a reversible checkpoint created by the harness.
type HarnessCheckpoint struct {
	ID        string  `json:"id,omitempty"`
	Handle    string  `json:"handle,omitempty"`
	CreatedAt float64 `json:"createdAt,omitempty"`
	Summary   string  `json:"summary,omitempty"`
}

// HarnessCheckResult reports the outcome of one verification check.
type HarnessCheckResult struct {
	Name    string `json:"name,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Target  string `json:"target,omitempty"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Passed  bool   `json:"passed,omitempty"`
}

// HarnessEvidence captures evidence emitted during observe/verify/apply.
type HarnessEvidence struct {
	Kind    string `json:"kind,omitempty"`
	Label   string `json:"label,omitempty"`
	Path    string `json:"path,omitempty"`
	Value   string `json:"value,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// HarnessCapabilitiesData describes the available harness surface.
type HarnessCapabilitiesData struct {
	Project           string   `json:"project,omitempty"`
	TDVersion         string   `json:"tdVersion,omitempty"`
	TDBuild           string   `json:"tdBuild,omitempty"`
	ConnectorName     string   `json:"connectorName,omitempty"`
	ConnectorVersion  string   `json:"connectorVersion,omitempty"`
	ProtocolVersion   int      `json:"protocolVersion,omitempty"`
	SupportedRoutes   []string `json:"supportedRoutes,omitempty"`
	SupportedFamilies []string `json:"supportedFamilies,omitempty"`
	Features          []string `json:"features,omitempty"`
	Constraints       []string `json:"constraints,omitempty"`
	Notes             []string `json:"notes,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
}

// HarnessObserveData describes the current state for the harness loop.
type HarnessObserveData struct {
	Scope          string             `json:"scope,omitempty"`
	Goal           string             `json:"goal,omitempty"`
	Summary        string             `json:"summary,omitempty"`
	RequestID      string             `json:"requestId,omitempty"`
	RollbackHandle string             `json:"rollbackHandle,omitempty"`
	Outputs        []HarnessTarget    `json:"outputs,omitempty"`
	Viewers        []HarnessTarget    `json:"viewers,omitempty"`
	Changes        []HarnessChange    `json:"changes,omitempty"`
	Hotspots       []HarnessMetric    `json:"hotspots,omitempty"`
	Warnings       []string           `json:"warnings,omitempty"`
	Errors         []string           `json:"errors,omitempty"`
	Checkpoint     *HarnessCheckpoint `json:"checkpoint,omitempty"`
}

// HarnessVerifyData reports the outcome of a verify step.
type HarnessVerifyData struct {
	Scope          string               `json:"scope,omitempty"`
	Goal           string               `json:"goal,omitempty"`
	Summary        string               `json:"summary,omitempty"`
	RequestID      string               `json:"requestId,omitempty"`
	RollbackHandle string               `json:"rollbackHandle,omitempty"`
	Passed         bool                 `json:"passed,omitempty"`
	Score          float64              `json:"score,omitempty"`
	Checks         []HarnessCheckResult `json:"checks,omitempty"`
	Evidence       []HarnessEvidence    `json:"evidence,omitempty"`
	Warnings       []string             `json:"warnings,omitempty"`
	Errors         []string             `json:"errors,omitempty"`
}

// HarnessApplyData reports the outcome of an apply step.
type HarnessApplyData struct {
	Scope          string             `json:"scope,omitempty"`
	Goal           string             `json:"goal,omitempty"`
	Summary        string             `json:"summary,omitempty"`
	RequestID      string             `json:"requestId,omitempty"`
	RollbackHandle string             `json:"rollbackHandle,omitempty"`
	Applied        bool               `json:"applied,omitempty"`
	DryRun         bool               `json:"dryRun,omitempty"`
	Changes        []HarnessChange    `json:"changes,omitempty"`
	Warnings       []string           `json:"warnings,omitempty"`
	Errors         []string           `json:"errors,omitempty"`
	Checkpoint     *HarnessCheckpoint `json:"checkpoint,omitempty"`
}

// HarnessRollbackData reports the outcome of a rollback step.
type HarnessRollbackData struct {
	Scope     string   `json:"scope,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Handle    string   `json:"handle,omitempty"`
	RequestID string   `json:"requestId,omitempty"`
	Restored  bool     `json:"restored,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// HarnessHistoryEntry summarizes one prior harness request.
type HarnessHistoryEntry struct {
	Timestamp      float64  `json:"timestamp,omitempty"`
	Route          string   `json:"route,omitempty"`
	Scope          string   `json:"scope,omitempty"`
	Goal           string   `json:"goal,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	RequestID      string   `json:"requestId,omitempty"`
	RollbackHandle string   `json:"rollbackHandle,omitempty"`
	Success        bool     `json:"success,omitempty"`
	DryRun         bool     `json:"dryRun,omitempty"`
	Passed         bool     `json:"passed,omitempty"`
	Applied        bool     `json:"applied,omitempty"`
	Restored       bool     `json:"restored,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

// HarnessHistoryData returns recent harness activity for the current scope.
type HarnessHistoryData struct {
	Scope   string                `json:"scope,omitempty"`
	Summary string                `json:"summary,omitempty"`
	Cursor  string                `json:"cursor,omitempty"`
	Entries []HarnessHistoryEntry `json:"entries,omitempty"`
}
