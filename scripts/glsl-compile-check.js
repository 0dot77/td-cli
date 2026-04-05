#!/usr/bin/env node
// PostToolUse hook: checks GLSL shader compile status after dat write
// Reads hook stdin JSON, extracts DAT path, checks if it's a shader DAT,
// then queries TD for compile warnings.

let data = '';
process.stdin.on('data', chunk => data += chunk);
process.stdin.on('end', () => {
  try {
    const input = JSON.parse(data);
    const cmd = input.tool_input?.command || '';

    // Match: td-cli dat write <path>
    const m = cmd.match(/td-cli.*dat\s+write\s+([\S]+)/);
    if (!m) process.exit(0);

    const datPath = m[1];
    // Only trigger for shader DATs
    if (!/pixel|vertex|glsl/i.test(datPath)) process.exit(0);

    // Derive GLSL TOP path from DAT path (remove _pixel or _vertex suffix)
    const glslPath = datPath.replace(/_(pixel|vertex)$/, '');

    const { execSync } = require('child_process');
    const tdcli = process.env.TDCLI_PATH || 'C:/Dev/td-cli/td-cli.exe';

    // Query TD for warnings on the GLSL TOP
    const execCmd = `${tdcli} exec "w = op('${glslPath}').warnings(); print('GLSL_CHECK:' + str(w))"`;
    const result = execSync(execCmd, {
      encoding: 'utf8',
      timeout: 10000,
      cwd: 'C:/Dev/td-cli'
    }).trim();

    // Extract the warnings line
    const warnLine = result.split('\n').find(l => l.startsWith('GLSL_CHECK:'));
    if (!warnLine) process.exit(0);

    const warnings = warnLine.replace('GLSL_CHECK:', '').trim();

    if (warnings && warnings.length > 0 && warnings !== '') {
      // Shader has compile issues — inject context back to Claude
      const output = {
        hookSpecificOutput: {
          hookEventName: 'PostToolUse',
          additionalContext: `⚠ GLSL compile warning for ${glslPath}: ${warnings}\nRun: td-cli exec "return op('${glslPath}').warnings()" to see details. Fix the shader before continuing.`
        }
      };
      console.log(JSON.stringify(output));
    } else {
      // Shader compiled OK — brief confirmation
      const output = {
        hookSpecificOutput: {
          hookEventName: 'PostToolUse',
          additionalContext: `GLSL shader ${glslPath} compiled successfully.`
        }
      };
      console.log(JSON.stringify(output));
    }
  } catch (e) {
    // If TD is not running or any error, silently exit
    process.exit(0);
  }
});
