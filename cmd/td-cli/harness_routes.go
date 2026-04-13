package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/0dot77/td-cli/internal/client"
	"github.com/0dot77/td-cli/internal/commands"
	"github.com/0dot77/td-cli/internal/protocol"
)

func runHarness(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		printHarnessUsage()
		return nil
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "capabilities":
		payload, err := buildHarnessPayload(protocol.HarnessCapabilitiesRequest{}, args)
		if err != nil {
			return err
		}
		return commands.HarnessCapabilities(c, payload, jsonOutput)

	case "observe":
		req, extra, err := parseHarnessObserveArgs(args)
		if err != nil {
			return err
		}
		payload, err := buildHarnessPayload(req, extra)
		if err != nil {
			return err
		}
		return commands.HarnessObserve(c, payload, jsonOutput)

	case "verify":
		req, extra, err := parseHarnessVerifyArgs(args)
		if err != nil {
			return err
		}
		payload, err := buildHarnessPayload(req, extra)
		if err != nil {
			return err
		}
		return commands.HarnessVerify(c, payload, jsonOutput)

	case "apply":
		req, extra, err := parseHarnessApplyArgs(args)
		if err != nil {
			return err
		}
		payload, err := buildHarnessPayload(req, extra)
		if err != nil {
			return err
		}
		return commands.HarnessApply(c, payload, jsonOutput)

	case "rollback":
		req, extra, err := parseHarnessRollbackArgs(args)
		if err != nil {
			return err
		}
		payload, err := buildHarnessPayload(req, extra)
		if err != nil {
			return err
		}
		return commands.HarnessRollback(c, payload, jsonOutput)

	case "history":
		req, extra, err := parseHarnessHistoryArgs(args)
		if err != nil {
			return err
		}
		payload, err := buildHarnessPayload(req, extra)
		if err != nil {
			return err
		}
		return commands.HarnessHistory(c, payload, jsonOutput)

	case "help", "--help", "-h":
		printHarnessUsage()
		return nil

	default:
		return fmt.Errorf("unknown harness subcommand: %s (use capabilities, observe, verify, apply, rollback, history)", sub)
	}
}

func parseHarnessObserveArgs(args []string) (protocol.HarnessObserveRequest, []string, error) {
	req := protocol.HarnessObserveRequest{Depth: 2}
	extra := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--depth":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness observe [path] [--depth N] [--include-snapshot] [--file payload.json] [--data <json>]")
			}
			req.Depth, _ = strconv.Atoi(args[i+1])
			i++
		case "--include-snapshot":
			req.IncludeSnapshot = true
		case "--file", "--data":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness observe [path] [--depth N] [--include-snapshot] [--file payload.json] [--data <json>]")
			}
			extra = append(extra, args[i], args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return req, nil, fmt.Errorf("unknown harness observe flag: %s", args[i])
			}
			if req.Path != "" {
				return req, nil, fmt.Errorf("usage: td-cli harness observe [path] [--depth N] [--include-snapshot] [--file payload.json] [--data <json>]")
			}
			req.Path = args[i]
		}
	}
	if req.Path == "" {
		req.Path = "/"
	}
	return req, extra, nil
}

func parseHarnessVerifyArgs(args []string) (protocol.HarnessVerifyRequest, []string, error) {
	req := protocol.HarnessVerifyRequest{Depth: 2}
	extra := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--depth":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness verify <path> [--assert <json>] [--depth N] [--include-observation] [--file payload.json] [--data <json>]")
			}
			req.Depth, _ = strconv.Atoi(args[i+1])
			i++
		case "--include-observation":
			req.IncludeObservation = true
		case "--assert":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness verify <path> [--assert <json>] [--depth N] [--include-observation] [--file payload.json] [--data <json>]")
			}
			assertion, err := parseHarnessAssertion(args[i+1])
			if err != nil {
				return req, nil, err
			}
			req.Assertions = append(req.Assertions, assertion)
			i++
		case "--file", "--data":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness verify <path> [--assert <json>] [--depth N] [--include-observation] [--file payload.json] [--data <json>]")
			}
			extra = append(extra, args[i], args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return req, nil, fmt.Errorf("unknown harness verify flag: %s", args[i])
			}
			if req.Path != "" {
				return req, nil, fmt.Errorf("usage: td-cli harness verify <path> [--assert <json>] [--depth N] [--include-observation] [--file payload.json] [--data <json>]")
			}
			req.Path = args[i]
		}
	}
	if req.Path == "" {
		return req, nil, fmt.Errorf("usage: td-cli harness verify <path> [--assert <json>] [--depth N] [--include-observation] [--file payload.json] [--data <json>]")
	}
	return req, extra, nil
}

func parseHarnessApplyArgs(args []string) (protocol.HarnessApplyRequest, []string, error) {
	req := protocol.HarnessApplyRequest{SnapshotDepth: 20, StopOnError: true}
	extra := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--goal":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
			}
			req.Goal = args[i+1]
			i++
		case "--note":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
			}
			req.Note = args[i+1]
			i++
		case "--snapshot-depth":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
			}
			req.SnapshotDepth, _ = strconv.Atoi(args[i+1])
			i++
		case "--continue-on-error":
			req.StopOnError = false
		case "--op":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
			}
			op, err := parseHarnessOperation(args[i+1])
			if err != nil {
				return req, nil, err
			}
			req.Operations = append(req.Operations, op)
			i++
		case "--file", "--data":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
			}
			extra = append(extra, args[i], args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return req, nil, fmt.Errorf("unknown harness apply flag: %s", args[i])
			}
			if req.TargetPath != "" {
				return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
			}
			req.TargetPath = args[i]
		}
	}
	if req.TargetPath == "" {
		return req, nil, fmt.Errorf("usage: td-cli harness apply <targetPath> [--goal <text>] [--note <text>] [--snapshot-depth N] [--continue-on-error] [--op <json>] [--file payload.json] [--data <json>]")
	}
	return req, extra, nil
}

func parseHarnessRollbackArgs(args []string) (protocol.HarnessRollbackRequest, []string, error) {
	req := protocol.HarnessRollbackRequest{}
	extra := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file", "--data":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness rollback <id> [--file payload.json] [--data <json>]")
			}
			extra = append(extra, args[i], args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return req, nil, fmt.Errorf("unknown harness rollback flag: %s", args[i])
			}
			if req.ID != "" {
				return req, nil, fmt.Errorf("usage: td-cli harness rollback <id> [--file payload.json] [--data <json>]")
			}
			req.ID = args[i]
		}
	}
	if req.ID == "" {
		return req, nil, fmt.Errorf("usage: td-cli harness rollback <id> [--file payload.json] [--data <json>]")
	}
	return req, extra, nil
}

func parseHarnessHistoryArgs(args []string) (protocol.HarnessHistoryRequest, []string, error) {
	req := protocol.HarnessHistoryRequest{Limit: 20}
	extra := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness history [--target <path>] [--limit N] [--file payload.json] [--data <json>]")
			}
			req.TargetPath = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness history [--target <path>] [--limit N] [--file payload.json] [--data <json>]")
			}
			req.Limit, _ = strconv.Atoi(args[i+1])
			i++
		case "--file", "--data":
			if i+1 >= len(args) {
				return req, nil, fmt.Errorf("usage: td-cli harness history [--target <path>] [--limit N] [--file payload.json] [--data <json>]")
			}
			extra = append(extra, args[i], args[i+1])
			i++
		default:
			return req, nil, fmt.Errorf("unknown harness history flag: %s", args[i])
		}
	}
	return req, extra, nil
}

func parseHarnessAssertion(raw string) (protocol.HarnessAssertion, error) {
	var assertion protocol.HarnessAssertion
	if err := json.Unmarshal([]byte(raw), &assertion); err != nil {
		return protocol.HarnessAssertion{}, fmt.Errorf("invalid --assert payload: %w", err)
	}
	return assertion, nil
}

func parseHarnessOperation(raw string) (protocol.HarnessOperation, error) {
	var op protocol.HarnessOperation
	if err := json.Unmarshal([]byte(raw), &op); err != nil {
		return protocol.HarnessOperation{}, fmt.Errorf("invalid --op payload: %w", err)
	}
	return op, nil
}

func buildHarnessPayload(base interface{}, args []string) (map[string]interface{}, error) {
	filePath := ""
	inlineJSON := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --file")
			}
			filePath = args[i+1]
			i++
		case "--data":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --data")
			}
			inlineJSON = args[i+1]
			i++
		default:
			return nil, fmt.Errorf("unknown harness payload flag: %s", args[i])
		}
	}

	payload := map[string]interface{}{}

	if filePath != "" {
		filePayload, err := readJSONObjectFile(filePath)
		if err != nil {
			return nil, err
		}
		mergeObjectMaps(payload, filePayload)
	}

	if inlineJSON != "" {
		inlinePayload, err := parseJSONObject(inlineJSON)
		if err != nil {
			return nil, err
		}
		mergeObjectMaps(payload, inlinePayload)
	}

	basePayload, err := marshalObjectMap(base)
	if err != nil {
		return nil, err
	}
	mergeObjectMaps(payload, basePayload)

	return payload, nil
}

func readJSONObjectFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return parseJSONObject(string(data))
}

func parseJSONObject(raw string) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	return payload, nil
}

func marshalObjectMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode request payload: %w", err)
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	return payload, nil
}

func mergeObjectMaps(dst, src map[string]interface{}) {
	for key, value := range src {
		if valueMap, ok := value.(map[string]interface{}); ok {
			if existing, ok := dst[key].(map[string]interface{}); ok {
				mergeObjectMaps(existing, valueMap)
				continue
			}
		}
		dst[key] = value
	}
}
