package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/td-cli/td-cli/internal/client"
	"github.com/td-cli/td-cli/internal/commands"
	"github.com/td-cli/td-cli/internal/discovery"
	"github.com/td-cli/td-cli/internal/docs"
	"github.com/td-cli/td-cli/internal/protocol"
)

var version = "0.1.0"

// fixMsysPath reverses Git Bash's automatic path conversion.
// Git Bash converts "/project1" to "C:/Program Files/Git/project1".
func fixMsysPath(s string) string {
	if runtime.GOOS != "windows" {
		return s
	}
	// Detect MSYS path conversion: "C:/Program Files/Git/" prefix
	gitPrefixes := []string{
		"C:/Program Files/Git/",
		"C:\\Program Files\\Git\\",
	}
	for _, prefix := range gitPrefixes {
		if strings.HasPrefix(s, prefix) {
			return "/" + s[len(prefix):]
		}
	}
	return s
}

func main() {
	rawArgs := os.Args[1:]
	args := make([]string, len(rawArgs))
	for i, a := range rawArgs {
		args[i] = fixMsysPath(a)
	}

	// Parse global flags
	var (
		port       int
		project    string
		jsonOutput bool
		timeout    = 30 * time.Second
	)

	// Extract global flags before the subcommand
	var cmdArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				port, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "--project":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		case "--json":
			jsonOutput = true
		case "--timeout":
			if i+1 < len(args) {
				if ms, err := strconv.Atoi(args[i+1]); err == nil {
					timeout = time.Duration(ms) * time.Millisecond
				}
				i++
			}
		default:
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	if len(cmdArgs) == 0 {
		printUsage()
		os.Exit(0)
	}

	command := cmdArgs[0]
	cmdArgs = cmdArgs[1:]

	switch command {
	case "version":
		fmt.Printf("td-cli v%s\n", version)

	case "help", "--help", "-h":
		printUsage()

	case "instances":
		instances, err := discovery.ScanInstances()
		if err != nil {
			fatal(err)
		}
		commands.Instances(instances, jsonOutput)

	case "update":
		if err := commands.Update(version, jsonOutput); err != nil {
			fatal(err)
		}

	case "init":
		if err := commands.Init(jsonOutput, port, 5000); err != nil {
			fatal(err)
		}

	case "context":
		c, err := getClient(port, project, timeout)
		if err != nil {
			fatal(err)
		}
		depth := 2
		for i := 0; i < len(cmdArgs); i++ {
			if cmdArgs[i] == "--depth" && i+1 < len(cmdArgs) {
				depth, _ = strconv.Atoi(cmdArgs[i+1])
				i++
			}
		}
		if err := commands.Context(c, depth, jsonOutput); err != nil {
			fatal(err)
		}

	case "docs":
		if err := runDocs(cmdArgs, jsonOutput); err != nil {
			fatal(err)
		}

	case "shaders":
		if err := runShaders(cmdArgs, jsonOutput, port, project, timeout); err != nil {
			fatal(err)
		}

	case "diff":
		if err := runDiff(cmdArgs, jsonOutput, port, project, timeout); err != nil {
			fatal(err)
		}

	default:
		// Commands that need a TD connection
		c, err := getClient(port, project, timeout)
		if err != nil {
			fatal(err)
		}
		if err := runCommand(c, command, cmdArgs, jsonOutput); err != nil {
			fatal(err)
		}
	}
}

func getClient(port int, project string, timeout time.Duration) (*client.Client, error) {
	inst, err := discovery.FindInstance(port, project)
	if err != nil {
		return nil, err
	}
	return client.New(inst.Port, timeout), nil
}

func runCommand(c *client.Client, command string, args []string, jsonOutput bool) error {
	switch command {
	case "status":
		return commands.Status(c, jsonOutput)

	case "exec":
		code, filePath, verifyPath, screenshotPath := parseExecArgs(args)
		return commands.Exec(c, code, filePath, jsonOutput, verifyPath, screenshotPath)

	case "ops":
		return runOps(c, args, jsonOutput)

	case "par":
		return runPar(c, args, jsonOutput)

	case "connect":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli connect <src> <dst> [--src-index N] [--dst-index N]")
		}
		srcIdx, dstIdx := 0, 0
		for i := 2; i < len(args); i++ {
			if args[i] == "--src-index" && i+1 < len(args) {
				srcIdx, _ = strconv.Atoi(args[i+1])
			}
			if args[i] == "--dst-index" && i+1 < len(args) {
				dstIdx, _ = strconv.Atoi(args[i+1])
			}
		}
		return commands.Connect(c, args[0], args[1], srcIdx, dstIdx, jsonOutput)

	case "disconnect":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli disconnect <src> <dst>")
		}
		return commands.Disconnect(c, args[0], args[1], jsonOutput)

	case "dat":
		return runDat(c, args, jsonOutput)

	case "screenshot":
		path := ""
		outputFile := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "-o" && i+1 < len(args) {
				outputFile = args[i+1]
				i++
			} else if path == "" {
				path = args[i]
			}
		}
		return commands.Screenshot(c, path, outputFile, jsonOutput)

	case "project":
		return runProject(c, args, jsonOutput)

	case "backup":
		return runBackup(c, args, jsonOutput)

	case "logs":
		return runLogs(c, args, jsonOutput)

	case "tools":
		if len(args) == 0 || args[0] == "list" {
			return commands.ToolsList(c, jsonOutput)
		}
		return fmt.Errorf("unknown tools subcommand: %s (use list)", args[0])

	case "tox":
		return runTox(c, args, jsonOutput)

	case "network":
		return runNetwork(c, args, jsonOutput)

	case "describe":
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}
		return commands.Describe(c, path, jsonOutput)

	case "watch":
		path := "/"
		interval := 1 * time.Second
		for i := 0; i < len(args); i++ {
			if args[i] == "--interval" && i+1 < len(args) {
				if ms, err := strconv.Atoi(args[i+1]); err == nil {
					interval = time.Duration(ms) * time.Millisecond
				}
				i++
			} else {
				path = args[i]
			}
		}
		return commands.Watch(c, path, interval, jsonOutput)

	case "chop":
		return runChop(c, args, jsonOutput)

	case "sop":
		return runSop(c, args, jsonOutput)

	case "pop":
		return runPop(c, args, jsonOutput)

	case "table":
		return runTable(c, args, jsonOutput)

	case "timeline":
		return runTimeline(c, args, jsonOutput)

	case "cook":
		return runCook(c, args, jsonOutput)

	case "ui":
		return runUi(c, args, jsonOutput)

	case "batch":
		return runBatch(c, args, jsonOutput)

	case "media":
		return runMedia(c, args, jsonOutput)

	case "harness":
		return runHarness(c, args, jsonOutput)

	default:
		return fmt.Errorf("unknown command: %s\nRun 'td-cli help' for usage", command)
	}
}

func runOps(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli ops <list|create|delete|info> [args]")
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "list":
		path := "/"
		depth := 1
		family := ""
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--depth":
				if i+1 < len(args) {
					depth, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--family":
				if i+1 < len(args) {
					family = args[i+1]
					i++
				}
			default:
				if path == "/" {
					path = args[i]
				}
			}
		}
		return commands.OpsList(c, path, depth, family, jsonOutput)

	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli ops create <type> <parent> [--name <name>] [--x N] [--y N]")
		}
		opType := args[0]
		parent := args[1]
		name := ""
		nodeX, nodeY := -1, -1
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--name":
				if i+1 < len(args) {
					name = args[i+1]
					i++
				}
			case "--x":
				if i+1 < len(args) {
					nodeX, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--y":
				if i+1 < len(args) {
					nodeY, _ = strconv.Atoi(args[i+1])
					i++
				}
			}
		}
		return commands.OpsCreate(c, opType, parent, name, nodeX, nodeY, jsonOutput)

	case "delete":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli ops delete <path>")
		}
		return commands.OpsDelete(c, args[0], jsonOutput)

	case "info":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli ops info <path>")
		}
		return commands.OpsInfo(c, args[0], jsonOutput)

	case "rename":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli ops rename <path> <new-name>")
		}
		return commands.OpsRename(c, args[0], args[1], jsonOutput)

	case "copy":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli ops copy <src> <parent> [--name <name>]")
		}
		name := ""
		nodeX, nodeY := -1, -1
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--name":
				if i+1 < len(args) {
					name = args[i+1]
					i++
				}
			case "--x":
				if i+1 < len(args) {
					nodeX, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--y":
				if i+1 < len(args) {
					nodeY, _ = strconv.Atoi(args[i+1])
					i++
				}
			}
		}
		return commands.OpsCopy(c, args[0], args[1], name, nodeX, nodeY, jsonOutput)

	case "move":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli ops move <src> <parent>")
		}
		return commands.OpsMove(c, args[0], args[1], jsonOutput)

	case "clone":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli ops clone <src> <parent> [--name <name>]")
		}
		name := ""
		nodeX, nodeY := -1, -1
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--name":
				if i+1 < len(args) {
					name = args[i+1]
					i++
				}
			case "--x":
				if i+1 < len(args) {
					nodeX, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--y":
				if i+1 < len(args) {
					nodeY, _ = strconv.Atoi(args[i+1])
					i++
				}
			}
		}
		return commands.OpsClone(c, args[0], args[1], name, nodeX, nodeY, jsonOutput)

	case "search":
		parent := "/"
		pattern := ""
		family := ""
		depth := 10
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--family":
				if i+1 < len(args) {
					family = args[i+1]
					i++
				}
			case "--depth":
				if i+1 < len(args) {
					depth, _ = strconv.Atoi(args[i+1])
					i++
				}
			default:
				if parent == "/" {
					parent = args[i]
				} else if pattern == "" {
					pattern = args[i]
				}
			}
		}
		return commands.OpsSearch(c, parent, pattern, family, depth, jsonOutput)

	default:
		return fmt.Errorf("unknown ops subcommand: %s (use list, create, delete, info, rename, copy, move, clone, search)", sub)
	}
}

func runPar(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli par <get|set> <op> [args]")
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "get":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli par get <op> [param_names...]")
		}
		path := args[0]
		names := args[1:]
		return commands.ParGet(c, path, names, jsonOutput)

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: td-cli par set <op> <name> <value> [<name> <value>...]")
		}
		path := args[0]
		params := make(map[string]interface{})
		for i := 1; i+1 < len(args); i += 2 {
			if v, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				params[args[i]] = v
			} else if args[i+1] == "true" {
				params[args[i]] = true
			} else if args[i+1] == "false" {
				params[args[i]] = false
			} else {
				params[args[i]] = args[i+1]
			}
		}
		return commands.ParSet(c, path, params, jsonOutput)

	case "pulse":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli par pulse <op> <name>")
		}
		return commands.ParPulse(c, args[0], args[1], jsonOutput)

	case "reset":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli par reset <op> [names...]")
		}
		return commands.ParReset(c, args[0], args[1:], jsonOutput)

	case "expr":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli par expr <op> <name> [expression]")
		}
		expr := ""
		if len(args) > 2 {
			expr = args[2]
		}
		return commands.ParExpr(c, args[0], args[1], expr, jsonOutput)

	case "export":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli par export <op>")
		}
		return commands.ParExport(c, args[0], jsonOutput)

	case "import":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli par import <op> <json>")
		}
		var params []interface{}
		if err := json.Unmarshal([]byte(args[1]), &params); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return commands.ParImport(c, args[0], params, jsonOutput)

	default:
		return fmt.Errorf("unknown par subcommand: %s (use get, set, pulse, reset, expr, export, import)", sub)
	}
}

func runDat(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli dat <read|write> <path> [args]")
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "read":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli dat read <path>")
		}
		return commands.DatRead(c, args[0], jsonOutput)

	case "write":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli dat write <path> <content> [-f <file>]")
		}
		path := args[0]
		content := ""
		filePath := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "-f" && i+1 < len(args) {
				filePath = args[i+1]
				i++
			} else {
				if content != "" {
					content += " "
				}
				content += args[i]
			}
		}
		return commands.DatWrite(c, path, content, filePath, jsonOutput)

	default:
		return fmt.Errorf("unknown dat subcommand: %s (use read, write)", sub)
	}
}

func runProject(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return commands.ProjectInfo(c, jsonOutput)
	}

	switch args[0] {
	case "info":
		return commands.ProjectInfo(c, jsonOutput)
	case "save":
		path := ""
		if len(args) > 1 {
			path = args[1]
		}
		return commands.ProjectSave(c, path, jsonOutput)
	default:
		return fmt.Errorf("unknown project subcommand: %s (use info, save)", args[0])
	}
}

func runBackup(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return commands.BackupList(c, 20, jsonOutput)
	}

	switch args[0] {
	case "list":
		limit := 20
		for i := 1; i < len(args); i++ {
			if args[i] == "--limit" && i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.BackupList(c, limit, jsonOutput)
	case "restore":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli backup restore <backup-id>")
		}
		return commands.BackupRestore(c, args[1], jsonOutput)
	default:
		return fmt.Errorf("unknown backup subcommand: %s (use list, restore)", args[0])
	}
}

func runLogs(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return commands.LogsTail(c, 20, jsonOutput)
	}

	switch args[0] {
	case "list":
		limit := 20
		for i := 1; i < len(args); i++ {
			if args[i] == "--limit" && i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.LogsList(c, limit, jsonOutput)
	case "tail":
		limit := 20
		for i := 1; i < len(args); i++ {
			if args[i] == "--limit" && i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.LogsTail(c, limit, jsonOutput)
	default:
		return fmt.Errorf("unknown logs subcommand: %s (use list, tail)", args[0])
	}
}

func runTox(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli tox <export|import> [args]")
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "export":
		compPath := ""
		outputFile := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "-o" && i+1 < len(args) {
				outputFile = args[i+1]
				i++
			} else if compPath == "" {
				compPath = args[i]
			}
		}
		if compPath == "" || outputFile == "" {
			return fmt.Errorf("usage: td-cli tox export <comp_path> -o <file.tox>")
		}
		return commands.ToxExport(c, compPath, outputFile, jsonOutput)

	case "import":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli tox import <file.tox> [parent_path] [--name <name>]")
		}
		toxPath := args[0]
		parentPath := "/project1"
		name := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--name" && i+1 < len(args) {
				name = args[i+1]
				i++
			} else {
				parentPath = args[i]
			}
		}
		return commands.ToxImport(c, toxPath, parentPath, name, jsonOutput)

	default:
		return fmt.Errorf("unknown tox subcommand: %s (use export, import)", sub)
	}
}

func runNetwork(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli network <export|import> [args]")
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "export":
		path := "/"
		outputFile := ""
		depth := 10
		includeDefaults := false
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-o":
				if i+1 < len(args) {
					outputFile = args[i+1]
					i++
				}
			case "--depth":
				if i+1 < len(args) {
					depth, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--include-defaults":
				includeDefaults = true
			default:
				path = args[i]
			}
		}
		return commands.NetworkExport(c, path, outputFile, depth, includeDefaults, jsonOutput)

	case "import":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli network import <file> [target_path]")
		}
		filePath := args[0]
		targetPath := "/"
		if len(args) > 1 {
			targetPath = args[1]
		}
		return commands.NetworkImport(c, filePath, targetPath, jsonOutput)

	default:
		return fmt.Errorf("unknown network subcommand: %s (use export, import)", sub)
	}
}

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

func runDocs(args []string, jsonOutput bool) error {
	if len(args) == 0 {
		// Show categories overview
		cats := docs.ListCategories()
		fmt.Println("TouchDesigner Documentation (629 operators, 69 Python API classes)")
		fmt.Println("\nOperator categories:")
		for cat, count := range cats {
			fmt.Printf("  %-8s %d operators\n", cat, count)
		}
		fmt.Println("\nUsage:")
		fmt.Println("  td-cli docs <operator>           Lookup operator (e.g., noise_top, noiseTOP)")
		fmt.Println("  td-cli docs search <keyword>     Search operators")
		fmt.Println("  td-cli docs search <kw> --cat TOP  Search within category")
		fmt.Println("  td-cli docs api <class>          Python API class (e.g., OP, CHOP, Par)")
		fmt.Println("  td-cli docs api                  List all Python API classes")
		return nil
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "search":
		if len(args) == 0 {
			return fmt.Errorf("usage: td-cli docs search <keyword> [--cat TOP|CHOP|SOP|DAT|COMP|MAT]")
		}
		query := args[0]
		category := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--cat" && i+1 < len(args) {
				category = args[i+1]
				i++
			}
		}
		results := docs.SearchOperators(query, category, 20)
		if len(results) == 0 {
			fmt.Printf("No operators found for '%s'\n", query)
			return nil
		}
		for _, r := range results {
			sum := r.Summary
			if len(sum) > 80 {
				sum = sum[:80] + "..."
			}
			fmt.Printf("  %-6s %-30s %s\n", r.Category, r.Name, sum)
		}
		fmt.Printf("\n%d result(s). Use 'td-cli docs <name>' for details.\n", len(results))
		return nil

	case "api":
		if len(args) == 0 {
			// List all API classes
			classes := docs.ListAPIClasses()
			fmt.Printf("Python API Classes (%d):\n", len(classes))
			for i, name := range classes {
				fmt.Printf("  %-20s", name)
				if (i+1)%4 == 0 {
					fmt.Println()
				}
			}
			fmt.Println()
			return nil
		}
		key, api := docs.LookupAPI(args[0])
		if api == nil {
			return fmt.Errorf("API class not found: %s", args[0])
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(api, "", "  ")
			fmt.Println(string(out))
		} else {
			_ = key
			fmt.Print(docs.FormatAPIClass(api))
		}
		return nil

	default:
		// Treat as operator lookup
		query := sub
		key, op := docs.LookupOperator(query)
		if op == nil {
			// Maybe they meant search
			results := docs.SearchOperators(query, "", 10)
			if len(results) > 0 {
				fmt.Printf("Operator '%s' not found. Did you mean:\n", query)
				for _, r := range results {
					fmt.Printf("  %-6s %s\n", r.Category, r.Name)
				}
				return nil
			}
			return fmt.Errorf("operator not found: %s", query)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(op, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Print(docs.FormatOperator(key, op))
		}
		return nil
	}
}

func parseExecArgs(args []string) (code, filePath, verifyPath, screenshotPath string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			filePath = args[i+1]
			i++
		} else if args[i] == "--verify" && i+1 < len(args) {
			verifyPath = args[i+1]
			i++
		} else if args[i] == "--screenshot" && i+1 < len(args) {
			screenshotPath = args[i+1]
			i++
		} else {
			if code != "" {
				code += " "
			}
			code += args[i]
		}
	}
	return
}

func printUsage() {
	usage := `td-cli v%s — TouchDesigner CLI for agents and artists

Usage: td-cli [flags] <command> [args]

Commands:
  status                         Check TD connection
  context                        Full project context (connection + network)
  instances                      List running TD instances
  exec <code>                    Execute Python in TD
  exec -f <file>                 Execute Python file
  exec ... --verify <path>       Verify node graph after exec
  exec ... --screenshot <path>   Screenshot TOP after exec
  ops list [path]                List operators
  ops create <type> <parent>     Create operator
  ops delete <path>              Delete operator
  ops info <path>                Operator details
  par get <op> [names...]        Get parameters
  par set <op> <n> <v> ...       Set parameters
  connect <src> <dst>            Connect operators
  disconnect <src> <dst>         Disconnect operators
  dat read <path>                Read DAT content
  dat write <path> <content>     Write DAT content
  screenshot [path] [-o file]    Capture TOP as PNG
  project info                   Project metadata
  project save [path]            Save project
  backup list [--limit N]        List recent backup artifacts
  backup restore <backup-id>     Restore a backup artifact
  logs list [--limit N]          List recent audit log events
  logs tail [--limit N]          Read recent audit log events
  tox export <comp> -o <file>    Export COMP as .tox file
  tox import <file> [parent]     Import .tox into project
  network export [path] [-o f]   Export network as JSON snapshot
  network import <file> [path]   Import network from snapshot
  describe [path]                AI-friendly network description
  harness <subcommand>           Agent harness surface
  diff <file1> <file2>           Compare two network snapshots
  diff --live <file> [path]      Compare snapshot vs live TD state
  watch [path] [--interval ms]   Real-time performance monitor
  tools list                     Discover available tools (AI agent discovery)
  harness capabilities           Show harness routes and features
  harness observe [path]         Capture agent-oriented state snapshot
  harness verify <path>          Run harness verification checks
  harness apply <path>           Apply a harness patch
  harness rollback <id>          Roll back a harness checkpoint
  harness history                Show recent harness activity
  timeline [info]                Timeline state
  timeline play                  Start playback
  timeline pause                 Pause playback
  timeline seek <time>           Seek to time
  timeline range --start --end   Set timeline range
  timeline rate <fps>            Set playback rate
  cook node <path>               Force cook operator
  cook network [path]            Force cook network
  ui navigate <path>             Navigate to operator
  ui select <path>               Select operator
  batch exec <file.json>         Batch execute commands
  batch parset <file.json>       Batch set parameters
  media info <path>              Media operator info
  media export <path> <file>     Export TOP as image
  media snapshot <path> [-o f]   Capture snapshot
  shaders list [--cat <cat>]     List shader templates
  shaders get <name>             Show shader template details
  shaders apply <name> <glsl>    Apply shader to GLSL TOP
  pop av [--root <path>]         Build POP audio visual scene
  docs <operator>                Operator documentation
  docs search <keyword>          Search operators
  docs api [class]               Python API reference
  init                           Generate CLAUDE.md + AGENTS.md
  update                         Self-update from GitHub Releases
  version                        Show version

Global Flags:
  --port <N>         Connect to specific port
  --project <path>   Target specific TD project
  --json             Output raw JSON
  --timeout <ms>     Request timeout (default: 30000)
`
	fmt.Printf(usage, version)
}

func printHarnessUsage() {
	fmt.Println(`Usage: td-cli harness <subcommand> [args]

Subcommands:
  capabilities                   Query supported harness routes/features
  observe [scope]                Capture an observation snapshot
  verify [scope]                 Run verification checks
  apply <scope>                  Apply a routed patch plan
  rollback <id>                  Restore a prior checkpoint
  history                        List recent harness iterations

Shared JSON Input:
  --file <payload.json>          Merge request payload from JSON file
  --data <json>                  Merge request payload from inline JSON object
  Explicit flags override values from --file/--data.

Examples:
  td-cli harness capabilities
  td-cli harness observe /project1 --depth 2 --include-snapshot
  td-cli harness verify /project1 --assert '{"kind":"family","equals":"COMP"}'
  td-cli harness apply /project1 --file patch.json
  td-cli harness rollback 1712900000-harness
  td-cli harness history --limit 10`)
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

func runDiff(args []string, jsonOutput bool, port int, project string, timeout time.Duration) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli diff <file1> <file2> | td-cli diff --live <snapshot> [path]")
	}

	live := false
	path := "/"
	var fileArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--live" {
			live = true
		} else if args[i] == "--path" && i+1 < len(args) {
			path = args[i+1]
			i++
		} else {
			fileArgs = append(fileArgs, args[i])
		}
	}

	if live {
		if len(fileArgs) < 1 {
			return fmt.Errorf("usage: td-cli diff --live <snapshot.json> [path]")
		}
		if len(fileArgs) > 1 {
			path = fileArgs[1]
		}
		c, err := getClient(port, project, timeout)
		if err != nil {
			return err
		}
		return commands.DiffLive(c, fileArgs[0], path, jsonOutput)
	}

	if len(fileArgs) < 2 {
		return fmt.Errorf("usage: td-cli diff <file1> <file2>")
	}
	return commands.DiffFiles(fileArgs[0], fileArgs[1], jsonOutput)
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

func runShaders(args []string, jsonOutput bool, port int, project string, timeout time.Duration) error {
	if len(args) == 0 {
		return commands.ShadersList("", jsonOutput)
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "list":
		category := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "--cat" && i+1 < len(args) {
				category = args[i+1]
				i++
			}
		}
		return commands.ShadersList(category, jsonOutput)

	case "get":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli shaders get <name>")
		}
		return commands.ShadersGet(args[0], jsonOutput)

	case "apply":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli shaders apply <name> <glsl_top_path>")
		}
		c, err := getClient(port, project, timeout)
		if err != nil {
			return err
		}
		return commands.ShadersApply(c, args[0], args[1], jsonOutput)

	default:
		return fmt.Errorf("unknown shaders subcommand: %s (use list, get, apply)", sub)
	}
}

func runChop(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli chop <info|channels|sample> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "info":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli chop info <path>")
		}
		return commands.ChopInfo(c, args[0], jsonOutput)
	case "channels":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli chop channels <path> [--start N] [--count N]")
		}
		start, count := 0, -1
		for i := 1; i < len(args); i++ {
			if args[i] == "--start" && i+1 < len(args) {
				start, _ = strconv.Atoi(args[i+1])
				i++
			} else if args[i] == "--count" && i+1 < len(args) {
				count, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.ChopChannels(c, args[0], start, count, jsonOutput)
	case "sample":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli chop sample <path> [--channel <name>] [--index N]")
		}
		channel := ""
		index := 0
		for i := 1; i < len(args); i++ {
			if args[i] == "--channel" && i+1 < len(args) {
				channel = args[i+1]
				i++
			} else if args[i] == "--index" && i+1 < len(args) {
				index, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.ChopSample(c, args[0], channel, index, jsonOutput)
	default:
		return fmt.Errorf("unknown chop subcommand: %s (use info, channels, sample)", sub)
	}
}

func runSop(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli sop <info|points|attribs> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "info":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli sop info <path>")
		}
		return commands.SopInfo(c, args[0], jsonOutput)
	case "points":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli sop points <path> [--start N] [--limit N]")
		}
		start, limit := 0, 100
		for i := 1; i < len(args); i++ {
			if args[i] == "--start" && i+1 < len(args) {
				start, _ = strconv.Atoi(args[i+1])
				i++
			} else if args[i] == "--limit" && i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.SopPoints(c, args[0], start, limit, jsonOutput)
	case "attribs":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli sop attribs <path>")
		}
		return commands.SopAttribs(c, args[0], jsonOutput)
	default:
		return fmt.Errorf("unknown sop subcommand: %s (use info, points, attribs)", sub)
	}
}

func runPop(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli pop <info|points|prims|verts|bounds|attributes|save|av> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "info":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop info <path>")
		}
		return commands.PopInfo(c, args[0], jsonOutput)
	case "points":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop points <path> [--attr P] [--start 0] [--count 100]")
		}
		attr, start, count := "P", 0, -1
		for i := 1; i < len(args); i++ {
			if args[i] == "--attr" && i+1 < len(args) {
				attr = args[i+1]
				i++
			} else if args[i] == "--start" && i+1 < len(args) {
				start, _ = strconv.Atoi(args[i+1])
				i++
			} else if args[i] == "--count" && i+1 < len(args) {
				count, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.PopPoints(c, args[0], attr, start, count, jsonOutput)
	case "prims":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop prims <path> [--attr N] [--start 0] [--count 100]")
		}
		attr, start, count := "N", 0, -1
		for i := 1; i < len(args); i++ {
			if args[i] == "--attr" && i+1 < len(args) {
				attr = args[i+1]
				i++
			} else if args[i] == "--start" && i+1 < len(args) {
				start, _ = strconv.Atoi(args[i+1])
				i++
			} else if args[i] == "--count" && i+1 < len(args) {
				count, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.PopPrims(c, args[0], attr, start, count, jsonOutput)
	case "verts":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop verts <path> [--attr uv] [--start 0] [--count 100]")
		}
		attr, start, count := "uv", 0, -1
		for i := 1; i < len(args); i++ {
			if args[i] == "--attr" && i+1 < len(args) {
				attr = args[i+1]
				i++
			} else if args[i] == "--start" && i+1 < len(args) {
				start, _ = strconv.Atoi(args[i+1])
				i++
			} else if args[i] == "--count" && i+1 < len(args) {
				count, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.PopVerts(c, args[0], attr, start, count, jsonOutput)
	case "bounds":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop bounds <path>")
		}
		return commands.PopBounds(c, args[0], jsonOutput)
	case "attributes":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop attributes <path>")
		}
		return commands.PopAttributes(c, args[0], jsonOutput)
	case "save":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli pop save <path> [-o file]")
		}
		filepath := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "-o" && i+1 < len(args) {
				filepath = args[i+1]
				i++
			}
		}
		return commands.PopSave(c, args[0], filepath, jsonOutput)
	case "av":
		root := "/project1"
		name := "pop_audio_visual"
		template := "audio-reactive"
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--root":
				if i+1 < len(args) {
					root = args[i+1]
					i++
				}
			case "--name":
				if i+1 < len(args) {
					name = args[i+1]
					i++
				}
			default:
				if !strings.HasPrefix(args[i], "--") {
					template = args[i]
				}
			}
		}
		return commands.PopAV(c, template, root, name, jsonOutput)
	default:
		return fmt.Errorf("unknown pop subcommand: %s (use info, points, prims, verts, bounds, attributes, save, av)", sub)
	}
}

func runTable(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli table <rows|cell|append|delete> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "rows":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli table rows <path> [--start N] [--end N]")
		}
		start, end := 0, -1
		for i := 1; i < len(args); i++ {
			if args[i] == "--start" && i+1 < len(args) {
				start, _ = strconv.Atoi(args[i+1])
				i++
			} else if args[i] == "--end" && i+1 < len(args) {
				end, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.TableRows(c, args[0], start, end, jsonOutput)
	case "cell":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli table cell <path> <row> <col> [--value V]")
		}
		row, col, value := commands.ParseTableCoords(args[1:])
		return commands.TableCell(c, args[0], row, col, value, jsonOutput)
	case "append":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli table append <path> [--row|--col] [--values v1,v2]")
		}
		mode := "row"
		var values []string
		for i := 1; i < len(args); i++ {
			if args[i] == "--col" {
				mode = "col"
			} else if args[i] == "--row" {
				mode = "row"
			} else if args[i] == "--values" && i+1 < len(args) {
				values = strings.Split(args[i+1], ",")
				i++
			}
		}
		return commands.TableAppend(c, args[0], mode, values, jsonOutput)
	case "delete":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli table delete <path> [--row|--col] [--index N]")
		}
		mode := "row"
		index := -1
		for i := 1; i < len(args); i++ {
			if args[i] == "--col" {
				mode = "col"
			} else if args[i] == "--row" {
				mode = "row"
			} else if args[i] == "--index" && i+1 < len(args) {
				index, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		return commands.TableDelete(c, args[0], mode, index, jsonOutput)
	default:
		return fmt.Errorf("unknown table subcommand: %s (use rows, cell, append, delete)", sub)
	}
}

func runTimeline(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return commands.TimelineInfo(c, jsonOutput)
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "info":
		return commands.TimelineInfo(c, jsonOutput)
	case "play":
		return commands.TimelinePlay(c, jsonOutput)
	case "pause":
		return commands.TimelinePause(c, jsonOutput)
	case "seek":
		timeVal := -1.0
		for i := 0; i < len(args); i++ {
			timeVal, _ = strconv.ParseFloat(args[i], 64)
		}
		if timeVal < 0 {
			return fmt.Errorf("usage: td-cli timeline seek <time>")
		}
		return commands.TimelineSeek(c, timeVal, jsonOutput)
	case "range":
		start, end := -1.0, -1.0
		for i := 0; i+1 < len(args); i += 2 {
			switch args[i] {
			case "--start":
				start, _ = strconv.ParseFloat(args[i+1], 64)
			case "--end":
				end, _ = strconv.ParseFloat(args[i+1], 64)
			}
		}
		return commands.TimelineRange(c, start, end, jsonOutput)
	case "rate":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli timeline rate <fps>")
		}
		rate, _ := strconv.ParseFloat(args[0], 64)
		return commands.TimelineRate(c, rate, jsonOutput)
	default:
		return fmt.Errorf("unknown timeline subcommand: %s (use info, play, pause, seek, range, rate)", sub)
	}
}

func runCook(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli cook <node|network> <path>")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "node":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli cook node <path>")
		}
		return commands.CookNode(c, args[0], jsonOutput)
	case "network":
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}
		return commands.CookNetwork(c, path, jsonOutput)
	default:
		return fmt.Errorf("unknown cook subcommand: %s (use node, network)", sub)
	}
}

func runUi(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli ui <navigate|select|pulse> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "navigate":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli ui navigate <path>")
		}
		return commands.UiNavigate(c, args[0], jsonOutput)
	case "select":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli ui select <path>")
		}
		return commands.UiSelect(c, args[0], jsonOutput)
	case "pulse":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli ui pulse <path> <param>")
		}
		return commands.UiPulse(c, args[0], args[1], jsonOutput)
	default:
		return fmt.Errorf("unknown ui subcommand: %s (use navigate, select, pulse)", sub)
	}
}

func runBatch(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli batch <exec|parset> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "exec":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli batch exec <json_file>")
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		var cmds []map[string]interface{}
		if err := json.Unmarshal(data, &cmds); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return commands.BatchExec(c, cmds, jsonOutput)
	case "parset":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli batch parset <json_file>")
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		var sets []map[string]interface{}
		if err := json.Unmarshal(data, &sets); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return commands.BatchParSet(c, sets, jsonOutput)
	default:
		return fmt.Errorf("unknown batch subcommand: %s (use exec, parset)", sub)
	}
}

func runMedia(c *client.Client, args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli media <info|export|record|snapshot> [args]")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "info":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli media info <path>")
		}
		return commands.MediaInfo(c, args[0], jsonOutput)
	case "export":
		if len(args) < 2 {
			return fmt.Errorf("usage: td-cli media export <path> <output_file>")
		}
		return commands.MediaExport(c, args[0], args[1], jsonOutput)
	case "record":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli media record <path> [--start S] [--end E]")
		}
		start, end := 0.0, 0.0
		for i := 1; i+1 < len(args); i += 2 {
			switch args[i] {
			case "--start":
				start, _ = strconv.ParseFloat(args[i+1], 64)
			case "--end":
				end, _ = strconv.ParseFloat(args[i+1], 64)
			}
		}
		return commands.MediaRecord(c, args[0], start, end, jsonOutput)
	case "snapshot":
		if len(args) < 1 {
			return fmt.Errorf("usage: td-cli media snapshot <path> [-o file]")
		}
		outputFile := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "-o" && i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		}
		return commands.MediaSnapshot(c, args[0], outputFile, jsonOutput)
	default:
		return fmt.Errorf("unknown media subcommand: %s (use info, export, record, snapshot)", sub)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", strings.TrimSpace(err.Error()))
	os.Exit(1)
}
