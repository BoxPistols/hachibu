# Hachibu statusLine script for Claude Code on Windows without Git Bash (Claude Code then runs the
# statusLine through PowerShell). It saves the JSON that Claude Code passes on stdin to
# %LOCALAPPDATA%\Hachibu\statusline.json, which Hachibu for Windows reads.
# With Git Bash installed, Claude Code runs the statusLine through Git Bash; use statusline.sh then.
#
# Pass an existing statusLine command as arguments to keep using it: it receives the same JSON,
# and its output becomes the status line.
#
# ~/.claude/settings.json (write the path with forward slashes):
#   "statusLine": {"type": "command", "command": "powershell -NoProfile -ExecutionPolicy Bypass -File C:/Users/you/.claude/hachibu-statusline.ps1"}
#
# This file is ASCII only on purpose: Windows PowerShell 5.1 reads a .ps1 without a BOM in the
# ANSI code page (Shift_JIS on Japanese Windows), which breaks non-ASCII comments.
$ErrorActionPreference = 'Stop'

$dir = if ($env:HACHIBU_DIR) { $env:HACHIBU_DIR } else { Join-Path $env:LOCALAPPDATA 'Hachibu' }
New-Item -ItemType Directory -Force -Path $dir | Out-Null

# Claude Code sends UTF-8. Reading it in the default ANSI code page would garble non-ASCII paths.
$utf8 = New-Object System.Text.UTF8Encoding $false
[Console]::InputEncoding = $utf8
$OutputEncoding = $utf8
$json = [Console]::In.ReadToEnd()

$tmp = Join-Path $dir ('.statusline.' + [Guid]::NewGuid().ToString('N'))
[System.IO.File]::WriteAllText($tmp, $json, $utf8)

# Pass the JSON this script received to the next command. Reading the shared file instead could
# hand it another session's JSON.
$status = 0
if ($args.Count -gt 0) {
    $rest = @()
    if ($args.Count -gt 1) { $rest = $args[1..($args.Count - 1)] }
    $json | & $args[0] @rest
    $status = $LASTEXITCODE
}

# A rename within the same folder, so a reader never sees a half-written file.
try {
    Move-Item -Force -LiteralPath $tmp -Destination (Join-Path $dir 'statusline.json')
} catch {
    Remove-Item -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
}
exit $status
