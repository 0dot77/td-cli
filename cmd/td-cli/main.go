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
		if err := commands.Context(c, jsonOutput); err != nil {
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
		code, filePath := parseExecArgs(args)
		return commands.Exec(c, code, filePath, jsonOutput)

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

func runDiff(args []string, jsonOutput bool, port int, project string, timeout time.Duration) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: td-cli diff <file1> <file2> | td-cli diff --live <snapshot> [path]")
	}

	// Check for --live mode
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
		// Only --live mode needs a TD connection
		c, err := getClient(port, project, timeout)
		if err != nil {
			return err
		}
		return commands.DiffLive(c, fileArgs[0], path, jsonOutput)
	}

	// Offline file-to-file comparison — no TD needed
	if len(fileArgs) < 2 {
		return fmt.Errorf("usage: td-cli diff <file1> <file2>")
	}
	return commands.DiffFiles(fileArgs[0], fileArgs[1], jsonOutput)
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

func parseExecArgs(args []string) (code, filePath string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			filePath = args[i+1]
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
  diff <file1> <file2>           Compare two network snapshots
  diff --live <file> [path]      Compare snapshot vs live TD state
  watch [path] [--interval ms]   Real-time performance monitor
  tools list                     Discover available tools (AI agent discovery)
  shaders list [--cat <cat>]     List shader templates
  shaders get <name>             Show shader template details
  shaders apply <name> <glsl>    Apply shader to GLSL TOP
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

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", strings.TrimSpace(err.Error()))
	os.Exit(1)
}
