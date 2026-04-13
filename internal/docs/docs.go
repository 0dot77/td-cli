package docs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Operator struct {
	Name     string    `json:"name"`
	Category string    `json:"cat"`
	Summary  string    `json:"sum"`
	Params   []Param   `json:"pars"`
	Tips     []string  `json:"tips"`
	Examples []string  `json:"ex"`
}

type Param struct {
	Name    string      `json:"n"`
	Label   string      `json:"l"`
	Group   string      `json:"g"`
	Default interface{} `json:"d"`
}

type APIClass struct {
	Name    string     `json:"name"`
	Desc    string     `json:"desc"`
	Members []Member   `json:"mem"`
	Methods []Method   `json:"met"`
}

type Member struct {
	Name string `json:"n"`
	Type string `json:"t"`
	RO   bool   `json:"ro"`
}

type Method struct {
	Name string `json:"n"`
	Sig  string `json:"s"`
	Ret  string `json:"r"`
}

var (
	operators map[string]Operator
	apiClasses map[string]APIClass
)

func init() {
	operators = make(map[string]Operator)
	json.Unmarshal(operatorsJSON, &operators)

	apiClasses = make(map[string]APIClass)
	json.Unmarshal(pythonAPIJSON, &apiClasses)
}

// normalizeTDName converts camelCase TD names like "noiseTOP" to underscore keys like "noise_top".
func normalizeTDName(name string) string {
	// Find where the suffix starts (TOP, CHOP, SOP, DAT, COMP, MAT, POP)
	suffixes := []string{"TOP", "CHOP", "SOP", "DAT", "COMP", "MAT", "POP"}
	lower := strings.ToLower(name)
	for _, suf := range suffixes {
		if strings.HasSuffix(name, suf) {
			prefix := lower[:len(lower)-len(suf)]
			return prefix + "_" + strings.ToLower(suf)
		}
	}
	return lower
}

// LookupOperator finds an operator by name (case-insensitive, partial match).
func LookupOperator(query string) (string, *Operator) {
	q := strings.ToLower(query)

	// Exact match first
	if op, ok := operators[q]; ok {
		return q, &op
	}

	// Try normalized form (e.g. "noiseTOP" -> "noise_top")
	normalized := normalizeTDName(query)
	if normalized != q {
		if op, ok := operators[normalized]; ok {
			return normalized, &op
		}
	}

	// Try with common suffixes
	for _, suffix := range []string{"_top", "_chop", "_sop", "_dat", "_comp", "_mat", "_pop"} {
		if op, ok := operators[q+suffix]; ok {
			key := q + suffix
			return key, &op
		}
	}

	// Partial match
	for key, op := range operators {
		if strings.Contains(key, q) {
			return key, &op
		}
	}

	return "", nil
}

// SearchOperators searches operators by keyword.
func SearchOperators(query string, category string, limit int) []SearchResult {
	q := strings.ToLower(query)
	var results []SearchResult

	for key, op := range operators {
		if category != "" && !strings.EqualFold(op.Category, category) {
			continue
		}

		score := 0
		nameLower := strings.ToLower(op.Name)
		keyLower := strings.ToLower(key)

		if strings.Contains(keyLower, q) {
			score += 10
		}
		if strings.Contains(nameLower, q) {
			score += 8
		}
		if strings.Contains(strings.ToLower(op.Summary), q) {
			score += 3
		}

		if score > 0 {
			results = append(results, SearchResult{
				Key:      key,
				Name:     op.Name,
				Category: op.Category,
				Summary:  op.Summary,
				Score:    score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

type SearchResult struct {
	Key      string
	Name     string
	Category string
	Summary  string
	Score    int
}

// LookupAPI finds a Python API class by name.
func LookupAPI(query string) (string, *APIClass) {
	q := strings.ToLower(query)

	for key, api := range apiClasses {
		if strings.EqualFold(key, q) || strings.EqualFold(api.Name, q) {
			return key, &api
		}
	}

	// Partial match
	for key, api := range apiClasses {
		if strings.Contains(strings.ToLower(key), q) || strings.Contains(strings.ToLower(api.Name), q) {
			return key, &api
		}
	}

	return "", nil
}

// ListCategories returns operator counts by category.
func ListCategories() map[string]int {
	cats := make(map[string]int)
	for _, op := range operators {
		cats[op.Category]++
	}
	return cats
}

// ListAPIClasses returns all Python API class names.
func ListAPIClasses() []string {
	var names []string
	for _, api := range apiClasses {
		names = append(names, api.Name)
	}
	sort.Strings(names)
	return names
}

// FormatOperator returns a human-readable string for an operator.
func FormatOperator(key string, op *Operator) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  (%s)\n", op.Name, op.Category)
	fmt.Fprintf(&b, "  %s\n", op.Summary)

	if len(op.Params) > 0 {
		fmt.Fprintf(&b, "\n  Parameters:\n")
		for _, p := range op.Params {
			def := ""
			if p.Default != nil {
				def = fmt.Sprintf(" (default: %v)", p.Default)
			}
			if p.Label != "" && p.Label != p.Name {
				fmt.Fprintf(&b, "    %-20s  %s%s\n", p.Name, p.Label, def)
			} else {
				fmt.Fprintf(&b, "    %-20s%s\n", p.Name, def)
			}
		}
	}

	if len(op.Tips) > 0 {
		fmt.Fprintf(&b, "\n  Tips:\n")
		for _, t := range op.Tips {
			fmt.Fprintf(&b, "    - %s\n", t)
		}
	}

	if len(op.Examples) > 0 {
		fmt.Fprintf(&b, "\n  Examples:\n")
		for _, ex := range op.Examples {
			fmt.Fprintf(&b, "    %s\n", ex)
		}
	}

	return b.String()
}

// FormatAPIClass returns a human-readable string for a Python API class.
func FormatAPIClass(api *APIClass) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", api.Name)
	fmt.Fprintf(&b, "  %s\n", api.Desc)

	if len(api.Members) > 0 {
		fmt.Fprintf(&b, "\n  Properties:\n")
		for _, m := range api.Members {
			ro := ""
			if m.RO {
				ro = " (read-only)"
			}
			fmt.Fprintf(&b, "    .%-25s  %s%s\n", m.Name, m.Type, ro)
		}
	}

	if len(api.Methods) > 0 {
		fmt.Fprintf(&b, "\n  Methods:\n")
		for _, m := range api.Methods {
			ret := ""
			if m.Ret != "" {
				ret = " -> " + m.Ret
			}
			fmt.Fprintf(&b, "    %s%s\n", m.Sig, ret)
		}
	}

	return b.String()
}
