package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/td-cli/td-cli/internal/client"
	"github.com/td-cli/td-cli/internal/discovery"
	"github.com/td-cli/td-cli/internal/protocol"
)

func Init(jsonOutput bool, port int, timeoutMs int) error {
	fmt.Println("Scanning for TouchDesigner instances...")

	var health *protocol.HealthData
	var portFound int
	c, err := tryConnect(port, timeoutMs)
	if err == nil {
		health, err = fetchHealth(c)
		if err == nil {
			portFound = c.Port()
		}
	}

	if health != nil {
		fmt.Printf("  Found: %s on port %d (TD %s)\n", health.Project, portFound, health.TDVersion)
	} else {
		fmt.Println("  No running TouchDesigner instance found.")
		fmt.Println("  (Files will be generated with generic instructions)")
	}

	if err := writeCLAUDEMD(health, portFound); err != nil {
		return err
	}
	fmt.Println("  Created CLAUDE.md")

	if err := writeAGENTSMD(health, portFound); err != nil {
		return err
	}
	fmt.Println("  Created AGENTS.md")

	if err := writeClaudeCommands(); err != nil {
		return err
	}
	fmt.Println("  Created .claude/commands/td-context.md")

	fmt.Println("\nSetup complete. Open your agent in this directory to start.")

	if jsonOutput {
		result := map[string]interface{}{
			"claudeMd":  true,
			"agentsMd":  true,
			"connected": health != nil,
		}
		if health != nil {
			result["project"] = health.Project
			result["port"] = portFound
			result["tdVersion"] = health.TDVersion
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	}

	return nil
}

func tryConnect(portOverride int, timeoutMs int) (*client.Client, error) {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	if portOverride > 0 {
		c := client.New(portOverride, timeout)
		_, err := c.Health()
		return c, err
	}

	instances, err := discovery.ScanInstances()
	if err == nil && len(instances) > 0 {
		inst := instances[0]
		c := client.New(inst.Port, timeout)
		if _, err := c.Health(); err == nil {
			return c, nil
		}
	}

	for _, p := range []int{9500, 9981, 8080, 8888, 9090} {
		c := client.New(p, 3*time.Second)
		if _, err := c.Health(); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("no TD instance found")
}

func fetchHealth(c *client.Client) (*protocol.HealthData, error) {
	resp, err := c.Health()
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("server error: %s", resp.Message)
	}
	var health protocol.HealthData
	if err := json.Unmarshal(resp.Data, &health); err != nil {
		return nil, err
	}
	return &health, nil
}

func writeCLAUDEMD(health *protocol.HealthData, port int) error {
	projectInfo := ""
	if health != nil {
		projectInfo = fmt.Sprintf("\n## Project: %s\n\n- TD Version: %s (build %s)\n- Port: %d\n",
			health.Project, health.TDVersion, health.TDBuild, port)
	}

	content := `# TouchDesigner Project - Agent Integration

## Auto-Discovery

This project uses ` + "`td-cli`" + ` to control TouchDesigner.
You do NOT need to ask the user to use td-cli — use it directly.

- Run ` + "`td-cli status`" + ` to check connection
- Run ` + "`td-cli context`" + ` for a full project overview
- Use td-cli commands directly for all TD operations
- If connection is lost, ask: "Is TouchDesigner running?"
` + projectInfo + `

## Quick Reference

` + "```" + `bash
# Check connection
td-cli status

# Full project context
td-cli context

# Execute Python in TouchDesigner
td-cli exec "return op('/project1').findChildren()"

# List operators
td-cli ops list /project1
td-cli ops list /project1 --depth 2 --family TOP

# Create an operator
td-cli ops create noiseTOP /project1 --name myNoise
td-cli ops create compositeTOP /project1 --name myComp --x 200 --y 0

# Operator management
td-cli ops info /project1/myNoise
td-cli ops rename /project1/myNoise myNoiseRenamed
td-cli ops copy /project1/myNoise /project1
td-cli ops delete /project1/myNoise

# Parameters
td-cli par get /project1/myNoise
td-cli par set /project1/myNoise rough 0.5 amp 1.0
td-cli par expr /project1/myNoise rough "me.time.frame"
td-cli par pulse /project1/button1 pulse

# Connect operators
td-cli connect /project1/myNoise /project1/out1
td-cli disconnect /project1/myNoise /project1/out1

# DAT content
td-cli dat read /project1/text1
td-cli dat write /project1/text1 "hello world"
td-cli dat write /project1/script1 -f myscript.py

# Table DAT
td-cli table rows /project1/table1
td-cli table cell /project1/table1 0 0
td-cli table append /project1/table1 --row

# CHOP data
td-cli chop info /project1/wave1
td-cli chop channels /project1/wave1

# SOP data
td-cli sop info /project1/sphere1
td-cli sop points /project1/sphere1 --limit 100

# POP data (GPU geometry)
td-cli pop info /project1/grid1
td-cli pop points /project1/grid1 --attr P --limit 100
td-cli pop bounds /project1/grid1

# Screenshot
td-cli screenshot /project1/out1 -o output.png

# Project
td-cli project info
td-cli project save

# Timeline
td-cli timeline status
td-cli timeline play
td-cli timeline pause

# Shaders
td-cli shaders list
td-cli shaders get fbm_noise
td-cli shaders apply fbm_noise /project1/glsl1

# Network snapshots
td-cli network export /project1 -o snapshot.json
td-cli network import snapshot.json /project1
td-cli describe /project1

# Documentation (offline)
td-cli docs noise_top
td-cli docs search render --cat TOP
td-cli docs api OP
` + "```" + `

## Connector Boundary

Treat ` + "`TDCliServer`" + ` as an installed connector, not as normal project code.
Do **not** rewrite ` + "`/project1/TDCliServer/*`" + ` unless explicitly developing the connector.

## Offline Documentation

` + "`td-cli docs`" + ` provides offline TouchDesigner documentation:

` + "```" + `bash
td-cli docs noise_top          # Operator lookup
td-cli docs search noise       # Search by keyword
td-cli docs search render --cat TOP
td-cli docs api OP             # Python API class
td-cli docs api                # List all API classes
` + "```" + `

## TouchDesigner Concepts
- Operators (OPs): TOP (textures), CHOP (channels), SOP (geometry, CPU),
  POP (geometry, GPU), DAT (data/text), COMP (containers), MAT (materials)
- Operators connect left-to-right (output -> input)
- Parameters control operator behavior
- Python scripting: ` + "`op('/path')`" + `, ` + "`me`" + `, ` + "`parent()`" + `
- Use ` + "`td-cli exec`" + ` to run arbitrary Python for anything not covered

## Operator Type Reference
Common types: noiseTOP, constantTOP, compositeTOP, moviefileinTOP,
textTOP, renderTOP, nullTOP, switchTOP, selectTOP, feedbackTOP,
geometryCOMP, cameraCOMP, lightCOMP, baseCOMP, containerCOMP,
waveCHOP, noiseCHOP, mathCHOP, nullCHOP, selectCHOP,
textDAT, tableDAT, scriptDAT, webDAT,
sphereSOP, boxSOP, gridSOP, noiseSOP, nullSOP,
gridPOP, spherePOP, noisePOP, transformPOP, nullPOP

## Global Flags
- ` + "`--port <N>`" + ` — connect to specific port (default: auto-discover)
- ` + "`--project <path>`" + ` — target specific TD project
- ` + "`--json`" + ` — output raw JSON
- ` + "`--timeout <ms>`" + ` — request timeout (default: 30000)

## Tips
- Use ` + "`td-cli context`" + ` to get an overview before starting work
- Use ` + "`--json`" + ` when you need structured output
- Operator paths are absolute (e.g., /project1/myOp)
- For ` + "`exec`" + `, prefix with ` + "`return`" + ` to get a value back
- Parameter names are abbreviated (e.g., ` + "`rough`" + ` not ` + "`roughness`" + `)
- New operators are auto-positioned to avoid overlap
`

	return os.WriteFile("CLAUDE.md", []byte(content), 0644)
}

func writeAGENTSMD(health *protocol.HealthData, port int) error {
	connectionBlock := ""
	if health != nil {
		connectionBlock = fmt.Sprintf(`## Auto-Detected Instance
- Port: %d
- Project: %s
- TD Version: %s (build %s)

`, port, health.Project, health.TDVersion, health.TDBuild)
	}

	content := `# TouchDesigner Agent Configuration
` + connectionBlock + `## Before any TD operation
` + "```" + `bash
td-cli status
` + "```" + `

## Getting Context
` + "```" + `bash
td-cli context    # Full project overview (connection + network summary)
` + "```" + `

## Convention
- Always use absolute paths: /project1/...
- Use --json for structured output when processing results
- Check status before mutations
- The connector (TDCliServer) is not to be modified
- Use td-cli exec for anything not covered by built-in commands
`

	return os.WriteFile("AGENTS.md", []byte(content), 0644)
}

func writeClaudeCommands() error {
	dir := ".claude/commands"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := `---
description: Get TouchDesigner project context and connection status
---
Run: td-cli context
`

	return os.WriteFile(filepath.Join(dir, "td-context.md"), []byte(content), 0644)
}
