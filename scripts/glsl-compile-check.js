#!/usr/bin/env node
// PostToolUse hook for GLSL shader DAT writes:
// 1. Compile check — query TD for warnings
// 2. Parameterization check — analyze source for hardcoded magic numbers

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
    if (!/pixel|vertex|glsl/i.test(datPath)) process.exit(0);

    const glslPath = datPath.replace(/_(pixel|vertex)$/, '');
    const { execSync } = require('child_process');
    const tdcli = process.env.TDCLI_PATH || 'C:/Dev/td-cli/td-cli.exe';

    const messages = [];

    // --- 1. Compile check ---
    try {
      const execCmd = `${tdcli} exec "w = op('${glslPath}').warnings(); print('GLSL_CHECK:' + str(w))"`;
      const result = execSync(execCmd, {
        encoding: 'utf8', timeout: 10000, cwd: 'C:/Dev/td-cli'
      }).trim();

      const warnLine = result.split('\n').find(l => l.startsWith('GLSL_CHECK:'));
      if (warnLine) {
        const warnings = warnLine.replace('GLSL_CHECK:', '').trim();
        if (warnings && warnings.length > 0 && warnings !== '') {
          messages.push(`⚠ COMPILE ERROR for ${glslPath}: ${warnings}`);
          messages.push(`Fix the shader before continuing.`);
        } else {
          messages.push(`✓ ${glslPath} compiled OK.`);
        }
      }
    } catch (e) {
      // TD not running — skip compile check
    }

    // --- 2. Parameterization check ---
    // Read the shader source that was just written
    try {
      const shaderSource = getShaderSource(cmd, input);
      if (shaderSource) {
        const issues = analyzeParameterization(shaderSource);
        if (issues.length > 0) {
          messages.push('');
          messages.push('⚠ PARAMETERIZATION issues:');
          issues.forEach(i => messages.push(`  - ${i}`));
          messages.push('');
          messages.push('Artists need uniforms to connect CHOP/audio/MIDI.');
          messages.push('Use GLSL TOP vec0-vec7 slots to expose parameters.');
        }
      }
    } catch (e) {
      // Ignore analysis errors
    }

    if (messages.length > 0) {
      const output = {
        hookSpecificOutput: {
          hookEventName: 'PostToolUse',
          additionalContext: messages.join('\n')
        }
      };
      console.log(JSON.stringify(output));
    }
  } catch (e) {
    process.exit(0);
  }
});

// Extract shader source from the command
function getShaderSource(cmd, input) {
  // If writing from file (-f flag), read the file
  const fileMatch = cmd.match(/-f\s+([\S]+)/);
  if (fileMatch) {
    try {
      const fs = require('fs');
      const path = require('path');
      let filePath = fileMatch[1];
      if (!path.isAbsolute(filePath)) {
        filePath = path.join('C:/Dev/td-cli', filePath);
      }
      return fs.readFileSync(filePath, 'utf8');
    } catch (e) {
      return null;
    }
  }

  // If writing inline content, extract from command
  // td-cli dat write <path> "content..."
  const inlineMatch = cmd.match(/td-cli.*dat\s+write\s+[\S]+\s+"([\s\S]+)"$/);
  if (inlineMatch) return inlineMatch[1];

  // Try from tool response
  if (input.tool_response?.stdout?.includes('Wrote content')) {
    // Content was written but we can't easily extract inline content
    // Try reading back from TD
    try {
      const datPath = cmd.match(/td-cli.*dat\s+write\s+([\S]+)/)[1];
      const tdcli = process.env.TDCLI_PATH || 'C:/Dev/td-cli/td-cli.exe';
      return execSync(`${tdcli} dat read ${datPath}`, {
        encoding: 'utf8', timeout: 5000, cwd: 'C:/Dev/td-cli'
      }).trim();
    } catch (e) {
      return null;
    }
  }

  return null;
}

// Analyze shader for parameterization issues
function analyzeParameterization(source) {
  const issues = [];
  const lines = source.split('\n');

  // Count uniform declarations
  const uniforms = lines.filter(l => /^\s*uniform\s+/.test(l) && !/uTD/.test(l));

  // Check for hardcoded colors (vec3 with literal RGB values in main/material functions)
  const colorPattern = /vec3\s*\(\s*(\d+\.\d+)\s*,\s*(\d+\.\d+)\s*,\s*(\d+\.\d+)\s*\)/g;
  let colorCount = 0;
  let match;
  while ((match = colorPattern.exec(source)) !== null) {
    const context = source.substring(Math.max(0, match.index - 50), match.index);
    // Skip common non-color uses (normals, directions, UVs)
    if (/normalize|cross|reflect|vec2|floor|fract|mod/i.test(context)) continue;
    // Skip 0,0,0 and 1,1,1
    if (match[1] === '0.0' && match[2] === '0.0' && match[3] === '0.0') continue;
    if (match[1] === '1.0' && match[2] === '1.0' && match[3] === '1.0') continue;
    colorCount++;
  }

  if (colorCount > 2 && uniforms.length < 3) {
    issues.push(`${colorCount} hardcoded colors found but only ${uniforms.length} uniforms declared. Expose colors as uniforms (uColor1, uColor2...).`);
  }

  // Check for hardcoded sizes/radii in SDF functions
  const sdfCallPattern = /sd(?:Sphere|Box|Torus|Cylinder)\s*\([^,]+,\s*(\d+\.\d+)/g;
  let sdfHardcoded = 0;
  while ((match = sdfCallPattern.exec(source)) !== null) {
    sdfHardcoded++;
  }
  if (sdfHardcoded > 0) {
    issues.push(`${sdfHardcoded} hardcoded SDF size/radius values. Use uniform (e.g., uShape.x) so artists can control geometry.`);
  }

  // Check for hardcoded animation speeds
  const speedPattern = /uTime\.x\s*\*\s*(\d+\.?\d*)/g;
  let speedCount = 0;
  while ((match = speedPattern.exec(source)) !== null) {
    speedCount++;
  }
  if (speedCount > 1 && !source.includes('uSpeed')) {
    issues.push(`${speedCount} hardcoded animation speeds (uTime.x * N). Add a uSpeed uniform for artist control.`);
  }

  // Check for hardcoded fog/camera distances
  if (/exp\s*\(\s*-\s*totalDist\s*\*\s*\d+\.\d+\s*\)/.test(source) && !source.includes('uCam') && !source.includes('uFog')) {
    issues.push('Hardcoded fog density. Use uniform (e.g., uCam.w or uFog.x) for atmosphere control.');
  }

  // General: too few uniforms for complex shader
  const lineCount = lines.filter(l => l.trim().length > 0 && !l.trim().startsWith('//')).length;
  if (lineCount > 40 && uniforms.length < 2) {
    issues.push(`Complex shader (${lineCount} lines) with only ${uniforms.length} uniform(s). Artists need more control points.`);
  }

  // Check if only uTime exists — at minimum need shape/color controls
  if (uniforms.length === 1 && /uTime/.test(source) && lineCount > 30) {
    issues.push('Only uTime uniform. Add at least uShape (geometry), uColor (appearance) for artist control.');
  }

  return issues;
}
