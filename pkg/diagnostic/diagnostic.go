package diagnostic

import (
	"encoding/json"
	"fmt"
)

type Diagnostic struct {
	Code           string `json:"code"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Col            int    `json:"col"`
	Node           string `json:"node,omitempty"`
	Message        string `json:"message"`
	Expected       string `json:"expected,omitempty"`
	Received       string `json:"received,omitempty"`
	SuggestedPatch string `json:"suggested_patch,omitempty"`
}

type Report struct {
	Status string       `json:"status"`
	Errors []Diagnostic `json:"errors"`
}

func NewReport(diags []Diagnostic) Report {
	status := "success"
	if len(diags) > 0 {
		status = "error"
	}
	return Report{
		Status: status,
		Errors: diags,
	}
}

func (r Report) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (d Diagnostic) String() string {
	msg := fmt.Sprintf("[%s] %s:%d:%d - %s", d.Code, d.File, d.Line, d.Col, d.Message)
	if d.SuggestedPatch != "" {
		msg += fmt.Sprintf(" (Sugerido: %s)", d.SuggestedPatch)
	}
	return msg
}
