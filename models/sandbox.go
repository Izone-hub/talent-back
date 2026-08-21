package models

type ExecutionType string

const (
	ExecutionTypeStandard  ExecutionType = "standard"
	ExecutionTypeFunction  ExecutionType = "function"
	ExecutionTypeFramework ExecutionType = "framework"
)

type ExecuteRequest struct {
	Language string        `json:"language"`
	Code     string        `json:"code"`
	Type     ExecutionType `json:"type"`
	Stdin    string        `json:"stdin,omitempty"`

	TemplateID string            `json:"template_id,omitempty"`
	Files      map[string]string `json:"files,omitempty"`

	TimeLimit   int `json:"time_limit,omitempty"`
	MemoryLimit int `json:"memory_limit,omitempty"`
}

type ExecuteResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	TimeMs   int64  `json:"timeMs"`
	Passed   *bool  `json:"passed,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ParseRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type ParsedFunction struct {
	Name       string   `json:"name"`
	Args       []string `json:"args"`
	StartLine  int      `json:"startLine"`
	EndLine    int      `json:"endLine"`
}

type ParseResponse struct {
	Functions []ParsedFunction `json:"functions"`
	Error     string           `json:"error,omitempty"`
}
