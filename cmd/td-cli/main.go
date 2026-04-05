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
		runInit()

	case "docs":
		if err := runDocs(cmdArgs, jsonOutput); err != nil {
			fatal(err)
		}

	case "shaders":
		if err := runShaders(cmdArgs, jsonOutput, port, project, timeout); err != nil {
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

	case "tools":
		if len(args) == 0 || args[0] == "list" {
			return commands.ToolsList(c, jsonOutput)
		}
		return fmt.Errorf("unknown tools subcommand: %s (use list)", args[0])

	case "network":
		return runNetwork(c, args, jsonOutput)

	case "describe":
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}
		return commands.Describe(c, path, jsonOutput)

	case "diff":
		return runDiff(c, args, jsonOutput)

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

	default:
		return fmt.Errorf("unknown ops subcommand: %s (use list, create, delete, info)", sub)
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
			// Try to parse as number
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

	default:
		return fmt.Errorf("unknown par subcommand: %s (use get, set)", sub)
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

func runDiff(c *client.Client, args []string, jsonOutput bool) error {
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
		return commands.DiffLive(c, fileArgs[0], path, jsonOutput)
	}

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

func runInit() {
	claudeMD := `# TouchDesigner Project - Claude Code Integration

## Available Tool: td-cli

This project is connected to a live TouchDesigner instance via ` + "`td-cli`" + `.
Use shell commands to control TouchDesigner.

### Quick Reference

` + "```" + `bash
# Check connection
td-cli status

# Execute Python in TouchDesigner
td-cli exec "print(op('/project1').findChildren())"

# List operators
td-cli ops list /project1
td-cli ops list /project1 --depth 2 --family TOP

# Create an operator (auto-positioned to avoid overlap)
td-cli ops create noiseTOP /project1 --name myNoise
td-cli ops create compositeTOP /project1 --name myComp --x 200 --y 0

# Get operator info
td-cli ops info /project1/myNoise

# Set parameters
td-cli par set /project1/myNoise rough 0.5 amp 1.0

# Get parameters
td-cli par get /project1/myNoise rough amp

# Connect operators
td-cli connect /project1/myNoise /project1/out1

# Disconnect operators
td-cli disconnect /project1/myNoise /project1/out1

# Read/write DAT content
td-cli dat read /project1/text1
td-cli dat write /project1/text1 "hello world"
td-cli dat write /project1/script1 -f myscript.py

# Screenshot (save TOP output as PNG)
td-cli screenshot /project1/out1 -o output.png

# Project info and save
td-cli project info
td-cli project save
` + "```" + `

### Live Code Editing

The td-cli server runs inside TouchDesigner as Python scripts in Text DATs.
You can **read and modify these scripts live** using ` + "`dat read`" + ` and ` + "`dat write`" + `:

` + "```" + `bash
# Read the server handler code
td-cli dat read /project1/TDCliServer/handler

# Update handler with a local file
td-cli dat write /project1/TDCliServer/handler -f td/td_cli_handler.py

# Read/update the webserver callbacks
td-cli dat read /project1/TDCliServer/webserver1_callbacks
td-cli dat write /project1/TDCliServer/webserver1_callbacks -f td/webserver_callbacks.py

# Read/update the heartbeat script
td-cli dat read /project1/TDCliServer/chopexec1
td-cli dat write /project1/TDCliServer/chopexec1 -f td/heartbeat.py
` + "```" + `

This means you can iterate on the server-side Python code without
manually editing inside TouchDesigner. Write the code locally, then
push it to TD with ` + "`dat write -f`" + `.

Any Text DAT or Script DAT in the project can be modified this way,
making it possible to develop TD Python scripts entirely from the terminal.

### Offline Documentation (629 operators + 69 Python API classes)

` + "`td-cli docs`" + ` provides offline TouchDesigner documentation lookup:

` + "```" + `bash
# Lookup an operator
td-cli docs noise_top
td-cli docs noiseTOP

# Search operators by keyword
td-cli docs search noise
td-cli docs search render --cat TOP

# Python API class reference
td-cli docs api OP
td-cli docs api Par
td-cli docs api CHOP

# List all API classes
td-cli docs api
` + "```" + `

Use this to look up parameter names, operator descriptions, and Python
API methods when building TD networks.

### TouchDesigner Concepts
- Operators (OPs) are the building blocks: TOP (textures), CHOP (channels),
  SOP (geometry), DAT (data/text), COMP (containers), MAT (materials)
- Operators connect left-to-right (output -> input)
- Parameters control operator behavior
- Python scripting via ` + "`td`" + ` module: ` + "`op('/path')`" + `, ` + "`me`" + `, ` + "`parent()`" + `
- Text DATs hold Python scripts — editable via ` + "`td-cli dat read/write`" + `
- Use ` + "`td-cli exec`" + ` to run arbitrary Python inside TD for anything not covered by built-in commands

### Operator Type Reference
Common types: noiseTOP, constantTOP, compositeTOP, moviefileinTOP,
textTOP, renderTOP, nullTOP, switchTOP, selectTOP, feedbackTOP,
geometryCOMP, cameraCOMP, lightCOMP, baseCOMP, containerCOMP,
waveCHOP, noiseCHOP, mathCHOP, nullCHOP, selectCHOP,
textDAT, tableDAT, scriptDAT, webDAT,
sphereSOP, boxSOP, gridSOP, noiseSOP, nullSOP

### Global Flags
- ` + "`--port <N>`" + ` — connect to specific port (default: auto-discover)
- ` + "`--project <path>`" + ` — target specific TD project
- ` + "`--json`" + ` — output raw JSON for parsing
- ` + "`--timeout <ms>`" + ` — request timeout (default: 30000)

### Tips
- Use ` + "`td-cli exec`" + ` for complex operations not covered by built-in commands
- Use ` + "`--json`" + ` flag when you need structured output to parse
- Operator paths are absolute (e.g., /project1/myOp)
- For ` + "`exec`" + `, prefix with ` + "`return`" + ` to get a value back
- Parameter names in TD are abbreviated (e.g., ` + "`rough`" + ` not ` + "`roughness`" + `) — use ` + "`par get`" + ` to discover actual names
- New operators are auto-positioned to avoid overlap; use ` + "`--x`" + `/` + "`--y`" + ` to override
`

	if err := os.WriteFile("CLAUDE.md", []byte(claudeMD), 0644); err != nil {
		fatal(fmt.Errorf("failed to write CLAUDE.md: %w", err))
	}
	fmt.Println("Created CLAUDE.md for Claude Code integration")
}

func printUsage() {
	usage := `td-cli v%s — TouchDesigner CLI for Claude Code

Usage: td-cli [flags] <command> [args]

Commands:
  status                         Check TD connection
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
  init                           Generate CLAUDE.md
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
