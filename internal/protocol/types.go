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
type HarnessCapabilitiesRequest struct{}

// HarnessObserveRequest requests an agent-oriented observation snapshot.
type HarnessObserveRequest struct {
	Path            string `json:"path,omitempty"`
	Depth           int    `json:"depth,omitempty"`
	IncludeSnapshot bool   `json:"includeSnapshot,omitempty"`
}

// HarnessAssertion describes one verification assertion.
type HarnessAssertion struct {
	Kind     string        `json:"kind,omitempty"`
	Name     string        `json:"name,omitempty"`
	Path     string        `json:"path,omitempty"`
	Equals   interface{}   `json:"equals,omitempty"`
	Min      interface{}   `json:"min,omitempty"`
	Max      interface{}   `json:"max,omitempty"`
	Contains string        `json:"contains,omitempty"`
	OneOf    []interface{} `json:"oneOf,omitempty"`
}

// HarnessVerifyRequest requests harness verification over a scope.
type HarnessVerifyRequest struct {
	Path               string             `json:"path,omitempty"`
	Depth              int                `json:"depth,omitempty"`
	IncludeObservation bool               `json:"includeObservation,omitempty"`
	Assertions         []HarnessAssertion `json:"assertions,omitempty"`
}

// HarnessOperation describes a routed harness operation.
type HarnessOperation struct {
	Route string                 `json:"route,omitempty"`
	Body  map[string]interface{} `json:"body,omitempty"`
}

// HarnessApplyRequest requests a patch/apply step.
type HarnessApplyRequest struct {
	TargetPath    string             `json:"targetPath,omitempty"`
	Goal          string             `json:"goal,omitempty"`
	Note          string             `json:"note,omitempty"`
	Iteration     int                `json:"iteration,omitempty"`
	SnapshotDepth int                `json:"snapshotDepth,omitempty"`
	StopOnError   bool               `json:"stopOnError,omitempty"`
	Operations    []HarnessOperation `json:"operations,omitempty"`
}

// HarnessRollbackRequest requests rollback of a prior harness change.
type HarnessRollbackRequest struct {
	ID string `json:"id,omitempty"`
}

// HarnessHistoryRequest requests recent harness activity.
type HarnessHistoryRequest struct {
	TargetPath string `json:"targetPath,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// HarnessCapabilitiesData describes the available harness surface.
type HarnessCapabilitiesData struct {
	Connector struct {
		Name            string `json:"name,omitempty"`
		Version         string `json:"version,omitempty"`
		ProtocolVersion int    `json:"protocolVersion,omitempty"`
		InstallMode     string `json:"installMode,omitempty"`
	} `json:"connector,omitempty"`
	Runtime struct {
		ProjectName string `json:"projectName,omitempty"`
		ProjectPath string `json:"projectPath,omitempty"`
		TDVersion   string `json:"tdVersion,omitempty"`
		TDBuild     string `json:"tdBuild,omitempty"`
		HarnessRoot string `json:"harnessRoot,omitempty"`
	} `json:"runtime,omitempty"`
	Tools struct {
		Count      int            `json:"count,omitempty"`
		Routes     []string       `json:"routes,omitempty"`
		Namespaces map[string]int `json:"namespaces,omitempty"`
	} `json:"tools,omitempty"`
	Support struct {
		Families    map[string][]string `json:"families,omitempty"`
		Rollback    bool                `json:"rollback,omitempty"`
		History     bool                `json:"history,omitempty"`
		Observe     bool                `json:"observe,omitempty"`
		Verify      bool                `json:"verify,omitempty"`
		BatchRoutes []string            `json:"batchRoutes,omitempty"`
	} `json:"support,omitempty"`
}

type HarnessGraphSummary struct {
	NodeCount       int            `json:"nodeCount,omitempty"`
	ConnectionCount int            `json:"connectionCount,omitempty"`
	Families        map[string]int `json:"families,omitempty"`
	DataFlow        []string       `json:"dataFlow,omitempty"`
}

type HarnessObserveData struct {
	Path            string                   `json:"path,omitempty"`
	Outputs         []map[string]interface{} `json:"outputs,omitempty"`
	RecentActivity  []map[string]interface{} `json:"recentActivity,omitempty"`
	SnapshotSummary map[string]interface{}   `json:"snapshotSummary,omitempty"`
	Graph           HarnessGraphSummary      `json:"graph,omitempty"`
	Project         map[string]interface{}   `json:"project,omitempty"`
	Target          map[string]interface{}   `json:"target,omitempty"`
	Issues          map[string]interface{}   `json:"issues,omitempty"`
	Performance     map[string]interface{}   `json:"performance,omitempty"`
}

type HarnessVerifyData struct {
	Path           string                   `json:"path,omitempty"`
	Passed         bool                     `json:"passed,omitempty"`
	AssertionCount int                      `json:"assertionCount,omitempty"`
	PassedCount    int                      `json:"passedCount,omitempty"`
	Evidence       map[string]interface{}   `json:"evidence,omitempty"`
	Assertions     []map[string]interface{} `json:"assertions,omitempty"`
}

type HarnessApplyData struct {
	RollbackID    string                   `json:"rollbackId,omitempty"`
	RecordPath    string                   `json:"recordPath,omitempty"`
	TargetPath    string                   `json:"targetPath,omitempty"`
	Status        string                   `json:"status,omitempty"`
	BeforeSummary map[string]interface{}   `json:"beforeSummary,omitempty"`
	AfterSummary  map[string]interface{}   `json:"afterSummary,omitempty"`
	Results       []map[string]interface{} `json:"results,omitempty"`
}

type HarnessRollbackData struct {
	RollbackID string                 `json:"rollbackId,omitempty"`
	RecordPath string                 `json:"recordPath,omitempty"`
	Restored   bool                   `json:"restored,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

type HarnessHistoryEntry struct {
	ID             string                 `json:"id,omitempty"`
	CreatedAt      float64                `json:"createdAt,omitempty"`
	UpdatedAt      float64                `json:"updatedAt,omitempty"`
	Status         string                 `json:"status,omitempty"`
	TargetPath     string                 `json:"targetPath,omitempty"`
	Goal           string                 `json:"goal,omitempty"`
	Iteration      interface{}            `json:"iteration,omitempty"`
	OperationCount int                    `json:"operationCount,omitempty"`
	FailureCount   int                    `json:"failureCount,omitempty"`
	RecordPath     string                 `json:"recordPath,omitempty"`
	RolledBackAt   interface{}            `json:"rolledBackAt,omitempty"`
	BeforeSummary  map[string]interface{} `json:"beforeSummary,omitempty"`
	AfterSummary   map[string]interface{} `json:"afterSummary,omitempty"`
}

type HarnessHistoryData struct {
	Iterations []HarnessHistoryEntry `json:"iterations,omitempty"`
}
