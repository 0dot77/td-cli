package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/0dot77/td-cli/internal/client"
	"github.com/0dot77/td-cli/internal/commands"
	"github.com/0dot77/td-cli/internal/discovery"
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
		debug      bool
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
		case "--debug":
			debug = true
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

	debugFlag = debug

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

	case "doctor":
		live := len(cmdArgs) > 0 && cmdArgs[0] == "live"
		if err := commands.Doctor(version, port, project, live, jsonOutput); err != nil {
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

// debugFlag is set by main() and read by getClient().
var debugFlag bool

func getClient(port int, project string, timeout time.Duration) (*client.Client, error) {
	inst, err := discovery.FindInstance(port, project)
	if err != nil {
		return nil, err
	}
	c := client.New(inst.Port, timeout)
	c.Debug = debugFlag
	return c, nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", strings.TrimSpace(err.Error()))
	os.Exit(1)
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
  exec ... --verify-strict       Fail if verify reports graph issues
  exec ... --screenshot <path>   Screenshot TOP after exec
  exec ... --screenshot <path> -o file.png
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
  doctor                         Diagnose setup and connection issues
  doctor live                    Also exercise live exec/screenshot/ui/observe routes
  update                         Self-update from GitHub Releases
  version                        Show version

Global Flags:
  --port <N>         Connect to specific port
  --project <path>   Target specific TD project
  --json             Output raw JSON
  --debug            Log HTTP requests/responses to stderr
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
