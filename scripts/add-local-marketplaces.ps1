#Requires -Version 7.0
<#
.SYNOPSIS
    Register the recommended Claude Code plugin marketplaces from the Wizard Kit.

.DESCRIPTION
    Wraps `claude plugin marketplace add` for each recommended marketplace.
    Skips marketplaces already present in
    ~/.claude/plugins/known_marketplaces.json. Reads the recommendation list
    from internal/kit/payload/templates/claude-code/plugin-marketplaces.json.template.

.PARAMETER DryRun
    Show which marketplaces would be added. No CLI calls.

.PARAMETER Force
    Re-add even if already registered.

.EXAMPLE
    pwsh -File scripts/add-local-marketplaces.ps1 -DryRun
    pwsh -File scripts/add-local-marketplaces.ps1
#>

[CmdletBinding()]
param(
    [switch] $DryRun,
    [switch] $Force
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$TemplatePath = Join-Path $RepoRoot 'internal/kit/payload/templates/claude-code/plugin-marketplaces.json.template'
$KnownPath = Join-Path $HOME '.claude/plugins/known_marketplaces.json'

function Write-Step([string] $m) { Write-Host "==> $m" -ForegroundColor Cyan }
function Write-Note([string] $m) { Write-Host "    $m" -ForegroundColor DarkGray }
function Write-Ok([string]   $m) { Write-Host "    + $m" -ForegroundColor Green }
function Write-Skip([string] $m) { Write-Host "    - $m" -ForegroundColor Yellow }

if (-not (Test-Path -LiteralPath $TemplatePath)) {
    throw "Template not found: $TemplatePath"
}

$tpl = Get-Content -LiteralPath $TemplatePath -Raw -Encoding utf8 | ConvertFrom-Json -Depth 50
$recommended = $tpl._recommended_entries.PSObject.Properties

$known = @{}
if (Test-Path -LiteralPath $KnownPath) {
    $raw = Get-Content -LiteralPath $KnownPath -Raw -Encoding utf8
    if (-not [string]::IsNullOrWhiteSpace($raw)) {
        $parsed = $raw | ConvertFrom-Json -Depth 50 -AsHashtable
        if ($parsed -is [System.Collections.IDictionary]) { $known = $parsed }
    }
}

$claudeCmd = Get-Command claude -ErrorAction SilentlyContinue
if (-not $claudeCmd -and -not $DryRun) {
    throw "`claude` CLI not found on PATH. Install Claude Code first, or run -DryRun."
}

Write-Step 'Wizard Kit: register local plugin marketplaces'
Write-Note "template: $TemplatePath"
Write-Note "known:    $KnownPath"
if ($DryRun) { Write-Note 'mode: DRY-RUN' }

foreach ($prop in $recommended) {
    $name = $prop.Name
    $entry = $prop.Value
    if ($known.ContainsKey($name) -and -not $Force) {
        Write-Skip "$name already registered"
        continue
    }

    $source = $entry.source
    $arg = if ($source.PSObject.Properties.Name -contains 'repo') { $source.repo } else { $name }

    if ($DryRun) {
        Write-Note "would run: claude plugin marketplace add $arg"
        continue
    }

    Write-Ok "claude plugin marketplace add $arg"
    & claude plugin marketplace add $arg
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "non-zero exit from `claude plugin marketplace add $arg` (code $LASTEXITCODE)"
    }
}

Write-Host ''
Write-Host 'Done.' -ForegroundColor Green
