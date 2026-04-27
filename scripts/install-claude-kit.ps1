#Requires -Version 7.0
<#
.SYNOPSIS
    Wizard Kit installer for Claude Code and OpenAI Codex CLI.

.DESCRIPTION
    Installs the Wizard Kit payload (skills, settings deltas, CLAUDE.md /
    AGENTS.md additions, plan/session indexes) from this repo into
    ~/.claude and ~/.codex. Idempotent. Backs up modified files into a
    timestamped folder under <target>/.wizard-kit-backup/<timestamp>/.

.PARAMETER DryRun
    Show what would change. No writes.

.PARAMETER Install
    Apply the kit. Default action when no other action flag is given.

.PARAMETER Rollback
    Restore the most recent backup for each enabled target.

.PARAMETER ClaudeOnly
    Skip ~/.codex.

.PARAMETER CodexOnly
    Skip ~/.claude.

.PARAMETER Force
    Suppress confirmation prompts. Required for non-interactive runs.

.EXAMPLE
    pwsh -File scripts/install-claude-kit.ps1 -DryRun
    pwsh -File scripts/install-claude-kit.ps1 -Install
    pwsh -File scripts/install-claude-kit.ps1 -Rollback -Force
    pwsh -File scripts/install-claude-kit.ps1 -Install -ClaudeOnly

.NOTES
    Reads payload from <repo-root>/internal/kit/payload/.
    Same payload is consumed by `ant kit install` (Go binary).
#>

[CmdletBinding(DefaultParameterSetName = 'Install')]
param(
    [Parameter(ParameterSetName = 'DryRun')]
    [switch] $DryRun,

    [Parameter(ParameterSetName = 'Install')]
    [switch] $Install,

    [Parameter(ParameterSetName = 'Rollback')]
    [switch] $Rollback,

    [switch] $ClaudeOnly,
    [switch] $CodexOnly,
    [switch] $Force
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$KitVersion = '0.1.0'
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$PayloadRoot = Join-Path $RepoRoot 'internal/kit/payload'
$Timestamp = (Get-Date -Format 'yyyyMMdd-HHmmss')

if (-not (Test-Path $PayloadRoot)) {
    throw "Payload not found at $PayloadRoot. Are you running from a clone of anthropic-cli?"
}

# Default to Install if no action chosen.
if (-not $DryRun -and -not $Install -and -not $Rollback) { $Install = $true }

if ($ClaudeOnly -and $CodexOnly) {
    throw 'Cannot pass -ClaudeOnly and -CodexOnly together.'
}

# ---- Helpers ---------------------------------------------------------------

function Write-Step([string] $Message) {
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Note([string] $Message) {
    Write-Host "    $Message" -ForegroundColor DarkGray
}

function Write-Change([string] $Message) {
    Write-Host "    + $Message" -ForegroundColor Green
}

function Write-Warn2([string] $Message) {
    Write-Host "    ! $Message" -ForegroundColor Yellow
}

function New-BackupFolder([string] $TargetDir) {
    $backupRoot = Join-Path $TargetDir '.wizard-kit-backup'
    $backupDir = Join-Path $backupRoot $Timestamp
    if (-not $DryRun) { New-Item -ItemType Directory -Force -Path $backupDir | Out-Null }
    return $backupDir
}

function Backup-File([string] $SourcePath, [string] $BackupDir, [string] $TargetDir) {
    if (-not (Test-Path -LiteralPath $SourcePath -PathType Leaf)) { return }
    $rel = [IO.Path]::GetRelativePath($TargetDir, $SourcePath)
    $dest = Join-Path $BackupDir $rel
    if (-not $DryRun) {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dest) | Out-Null
        Copy-Item -LiteralPath $SourcePath -Destination $dest -Force
    }
    Write-Note "backup: $rel"
}

function Test-IsDict($Value) {
    return ($null -ne $Value) -and ($Value -is [System.Collections.IDictionary])
}

function Test-IsList($Value) {
    if ($null -eq $Value) { return $false }
    if ($Value -is [string]) { return $false }
    if ($Value -is [System.Collections.IDictionary]) { return $false }
    return ($Value -is [System.Collections.IList]) -or ($Value -is [System.Array])
}

function Read-JsonFile([string] $Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    $raw = Get-Content -LiteralPath $Path -Raw -Encoding utf8
    if ([string]::IsNullOrWhiteSpace($raw)) { return $null }
    return $raw | ConvertFrom-Json -Depth 100 -AsHashtable
}

function Write-JsonFile([string] $Path, $Data) {
    $json = $Data | ConvertTo-Json -Depth 100
    if ($DryRun) { return }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Path) | Out-Null
    Set-Content -LiteralPath $Path -Value $json -Encoding utf8 -NoNewline
}

function Strip-WizardKitMeta($Data) {
    if (-not (Test-IsDict $Data)) { return $Data }
    $copy = [ordered]@{}
    foreach ($k in @($Data.Keys)) {
        if ($k -eq '_wizard_kit' -or $k -like '_wizard_kit_*') { continue }
        $v = $Data[$k]
        if (Test-IsDict $v) {
            $copy[$k] = Strip-WizardKitMeta $v
        } elseif (Test-IsList $v) {
            $copy[$k] = @(foreach ($e in $v) { if (Test-IsDict $e) { Strip-WizardKitMeta $e } else { $e } })
        } else {
            $copy[$k] = $v
        }
    }
    return $copy
}

function Merge-Dict($Existing, $Delta) {
    if (-not (Test-IsDict $Existing)) { $Existing = [ordered]@{} }
    foreach ($k in @($Delta.Keys)) {
        if ($k -eq '_wizard_kit' -or $k -like '_wizard_kit_*') { continue }
        $dv = $Delta[$k]
        if ($Existing.Contains($k)) {
            $ev = $Existing[$k]
            if ((Test-IsDict $ev) -and (Test-IsDict $dv)) {
                $Existing[$k] = Merge-Dict $ev $dv
            }
            elseif ((Test-IsList $ev) -and (Test-IsList $dv)) {
                $Existing[$k] = Merge-ArrayUnion $ev $dv
            }
            else {
                $Existing[$k] = $dv
            }
        }
        else {
            $Existing[$k] = $dv
        }
    }
    return $Existing
}

function Merge-ArrayUnion($Existing, $Delta) {
    $out = New-Object System.Collections.Generic.List[object]
    foreach ($e in @($Existing)) { [void] $out.Add($e) }

    foreach ($d in @($Delta)) {
        $isDict = Test-IsDict $d
        $hasId = $isDict -and $d.Contains('wizard_kit_id')
        if ($hasId) {
            $id = $d['wizard_kit_id']
            $idx = -1
            for ($i = 0; $i -lt $out.Count; $i++) {
                $existing = $out[$i]
                if ((Test-IsDict $existing) -and $existing.Contains('wizard_kit_id') -and $existing['wizard_kit_id'] -eq $id) {
                    $idx = $i; break
                }
            }
            if ($idx -ge 0) { $out[$idx] = $d } else { [void] $out.Add($d) }
        }
        else {
            $serial = ($d | ConvertTo-Json -Compress -Depth 50)
            $exists = $false
            foreach ($e in $out) {
                if (($e | ConvertTo-Json -Compress -Depth 50) -eq $serial) { $exists = $true; break }
            }
            if (-not $exists) { [void] $out.Add($d) }
        }
    }
    return ,$out.ToArray()
}

function Update-ManifestEntry([string] $TargetDir, [hashtable] $ManifestEntry) {
    $manifestPath = Join-Path $TargetDir '.wizard-kit-manifest.json'
    if ($DryRun) {
        Write-Note "would update manifest: $manifestPath"
        return
    }
    $manifest = if (Test-Path -LiteralPath $manifestPath) {
        Get-Content -LiteralPath $manifestPath -Raw -Encoding utf8 | ConvertFrom-Json -Depth 50 -AsHashtable
    } else {
        @{ history = @() }
    }
    if (-not $manifest.ContainsKey('history')) { $manifest['history'] = @() }
    $manifest['history'] = @($manifest['history']) + @($ManifestEntry)
    $manifest['last'] = $ManifestEntry
    Set-Content -LiteralPath $manifestPath -Value ($manifest | ConvertTo-Json -Depth 50) -Encoding utf8
}

# ---- Markdown delimited-section merge --------------------------------------

function Update-MarkdownSection {
    param(
        [string] $TargetPath,
        [string] $MarkerName,        # e.g. 'wizard-kit:claude-md'
        [string] $NewContent,
        [string] $BackupDir,
        [string] $TargetDir
    )
    $beginMarker = "<!-- $MarkerName BEGIN"
    $endMarker = "<!-- $MarkerName END -->"

    $existing = ''
    if (Test-Path -LiteralPath $TargetPath -PathType Leaf) {
        Backup-File -SourcePath $TargetPath -BackupDir $BackupDir -TargetDir $TargetDir
        $existing = Get-Content -LiteralPath $TargetPath -Raw -Encoding utf8
    }

    # Strip any prior managed block.
    $rx = [regex]::new(
        "(?ms)<!--\s$MarkerName\sBEGIN.*?<!--\s$MarkerName\sEND\s-->",
        'Multiline,Singleline'
    )
    $stripped = $rx.Replace($existing, '').TrimEnd()

    $newBlock = $NewContent.TrimEnd()
    $merged = if ([string]::IsNullOrWhiteSpace($stripped)) {
        $newBlock + "`n"
    } else {
        $stripped + "`n`n" + $newBlock + "`n"
    }

    if ($DryRun) {
        Write-Note "would update markdown section [$MarkerName] in $TargetPath"
    } else {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $TargetPath) | Out-Null
        Set-Content -LiteralPath $TargetPath -Value $merged -Encoding utf8 -NoNewline
    }
    Write-Change "merge markdown $TargetPath ($MarkerName)"
}

# ---- TOML "section append" merge (simple, idempotent) ----------------------

function Update-TomlSection {
    param(
        [string] $TargetPath,
        [string] $NewBody,
        [string] $BackupDir,
        [string] $TargetDir
    )
    $beginLine = '# === wizard-kit:codex BEGIN ==='
    $endLine = '# === wizard-kit:codex END ==='

    $existing = ''
    if (Test-Path -LiteralPath $TargetPath -PathType Leaf) {
        Backup-File -SourcePath $TargetPath -BackupDir $BackupDir -TargetDir $TargetDir
        $existing = Get-Content -LiteralPath $TargetPath -Raw -Encoding utf8
    }

    $rx = [regex]::new(
        "(?ms)# === wizard-kit:codex BEGIN ===.*?# === wizard-kit:codex END ===",
        'Multiline,Singleline'
    )
    $stripped = $rx.Replace($existing, '').TrimEnd()

    $block = "$beginLine`n$($NewBody.TrimEnd())`n$endLine"
    $merged = if ([string]::IsNullOrWhiteSpace($stripped)) {
        $block + "`n"
    } else {
        $stripped + "`n`n" + $block + "`n"
    }

    if ($DryRun) {
        Write-Note "would update TOML section in $TargetPath"
    } else {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $TargetPath) | Out-Null
        Set-Content -LiteralPath $TargetPath -Value $merged -Encoding utf8 -NoNewline
    }
    Write-Change "merge toml $TargetPath"
}

# ---- Plan / session index builders -----------------------------------------

function Get-FirstHeading([string] $Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    $lines = Get-Content -LiteralPath $Path -TotalCount 30 -Encoding utf8 -ErrorAction SilentlyContinue
    foreach ($line in $lines) {
        if ($line -match '^\s*#{1,3}\s+(.+)$') { return $Matches[1].Trim() }
    }
    return $null
}

function Build-PlanIndex([string] $PlansDir) {
    if (-not (Test-Path -LiteralPath $PlansDir -PathType Container)) { return '' }
    $plans = Get-ChildItem -LiteralPath $PlansDir -Filter '*.md' -File |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 60
    $lines = New-Object System.Collections.Generic.List[string]
    foreach ($p in $plans) {
        $title = Get-FirstHeading $p.FullName
        if (-not $title) { $title = $p.BaseName -replace '-', ' ' }
        $date = $p.LastWriteTime.ToString('yyyy-MM-dd')
        $rel = "../plans/$($p.Name)"
        $lines.Add("- [$title]($rel) — $date") | Out-Null
    }
    return [string]::Join("`n", $lines)
}

function Build-SessionIndex([string] $SessionsDir) {
    if (-not (Test-Path -LiteralPath $SessionsDir -PathType Container)) { return '' }
    $sessions = Get-ChildItem -LiteralPath $SessionsDir -Filter '*.jsonl' -File |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 40
    $lines = New-Object System.Collections.Generic.List[string]
    foreach ($s in $sessions) {
        $firstLine = Get-Content -LiteralPath $s.FullName -TotalCount 1 -Encoding utf8 -ErrorAction SilentlyContinue
        $topic = '(unparseable)'
        if ($firstLine) {
            try {
                $obj = $firstLine | ConvertFrom-Json -Depth 10
                if ($obj.PSObject.Properties.Name -contains 'prompt') { $topic = ($obj.prompt -replace '\s+', ' ').Substring(0, [Math]::Min(80, $obj.prompt.Length)) }
                elseif ($obj.PSObject.Properties.Name -contains 'message') { $topic = ($obj.message -replace '\s+', ' ').Substring(0, [Math]::Min(80, $obj.message.Length)) }
                elseif ($obj.PSObject.Properties.Name -contains 'content') { $topic = ($obj.content -replace '\s+', ' ').Substring(0, [Math]::Min(80, $obj.content.Length)) }
            } catch {}
        }
        $stamp = $s.LastWriteTime.ToString('yyyy-MM-dd HH:mm')
        $rel = "../sessions/$($s.Name)"
        $lines.Add("- [$topic]($rel) — $stamp") | Out-Null
    }
    return [string]::Join("`n", $lines)
}

function Update-IndexBlock {
    param(
        [string] $TargetPath,
        [string] $MarkerName,
        [string] $IndexBody
    )
    if (-not (Test-Path -LiteralPath $TargetPath -PathType Leaf)) {
        Write-Warn2 "index target missing, skipping: $TargetPath"
        return
    }
    $existing = Get-Content -LiteralPath $TargetPath -Raw -Encoding utf8

    $beginRx = "<!--\s$MarkerName\sBEGIN[^>]*-->"
    $endRx = "<!--\s$MarkerName\sEND\s-->"
    $blockRx = "(?ms)$beginRx.*?$endRx"

    $newBlock = "<!-- $MarkerName BEGIN — regenerated $(Get-Date -Format 'yyyy-MM-dd HH:mm') -->`n$IndexBody`n<!-- $MarkerName END -->"

    if ($existing -match $blockRx) {
        $merged = [regex]::Replace($existing, $blockRx, [System.Text.RegularExpressions.Regex]::Escape($newBlock).Replace('\$', '$$$$'), 'Singleline')
        # Use callback to avoid re-escaping bugs.
        $merged = [regex]::Replace(
            $existing,
            $blockRx,
            { param($m) $newBlock },
            'Singleline'
        )
    } else {
        $merged = $existing.TrimEnd() + "`n`n" + $newBlock + "`n"
    }

    if ($DryRun) {
        Write-Note "would update index block [$MarkerName] in $TargetPath"
    } else {
        Set-Content -LiteralPath $TargetPath -Value $merged -Encoding utf8 -NoNewline
    }
    Write-Change "rebuild index [$MarkerName] in $TargetPath"
}

# ---- Skill copy ------------------------------------------------------------

function Copy-Skills([string] $PayloadSkillRoot, [string] $TargetSkillRoot, [string] $BackupDir, [string] $TargetDir) {
    if (-not (Test-Path -LiteralPath $PayloadSkillRoot -PathType Container)) {
        Write-Warn2 "payload skill root missing: $PayloadSkillRoot"
        return @()
    }
    $installed = New-Object System.Collections.Generic.List[string]
    Get-ChildItem -LiteralPath $PayloadSkillRoot -Directory | ForEach-Object {
        $skillName = $_.Name
        $destDir = Join-Path $TargetSkillRoot $skillName
        $skillFile = Join-Path $_.FullName 'SKILL.md'
        if (-not (Test-Path -LiteralPath $skillFile)) {
            Write-Warn2 "skill missing SKILL.md: $skillName"
            return
        }
        $destFile = Join-Path $destDir 'SKILL.md'
        if (Test-Path -LiteralPath $destFile) {
            Backup-File -SourcePath $destFile -BackupDir $BackupDir -TargetDir $TargetDir
        }
        if (-not $DryRun) {
            New-Item -ItemType Directory -Force -Path $destDir | Out-Null
            Copy-Item -LiteralPath $skillFile -Destination $destFile -Force
        }
        Write-Change "skill $skillName -> $destFile"
        $installed.Add([IO.Path]::GetRelativePath($TargetDir, $destFile)) | Out-Null
    }
    return $installed.ToArray()
}

# ---- Settings.json merge ---------------------------------------------------

function Merge-SettingsJson([string] $TargetPath, [string] $TemplatePath, [string] $BackupDir, [string] $TargetDir) {
    if (-not (Test-Path -LiteralPath $TemplatePath -PathType Leaf)) {
        Write-Warn2 "template missing: $TemplatePath"
        return
    }
    if (Test-Path -LiteralPath $TargetPath) {
        Backup-File -SourcePath $TargetPath -BackupDir $BackupDir -TargetDir $TargetDir
    }
    $existing = Read-JsonFile $TargetPath
    if ($null -eq $existing) { $existing = [ordered]@{} }
    $delta = Read-JsonFile $TemplatePath
    if ($null -eq $delta) {
        Write-Warn2 "template empty: $TemplatePath"
        return
    }
    $delta = Strip-WizardKitMeta $delta
    $merged = Merge-Dict $existing $delta
    Write-JsonFile $TargetPath $merged
    Write-Change "merge settings.json -> $TargetPath"
}

# ---- Per-project local settings dropper ------------------------------------

function Drop-PerProjectLocal([string] $TargetClaudeDir) {
    # This walks the user's known project folders and offers to drop the
    # matching settings.local.json. Keep conservative: only known marker files
    # and only if no settings.local.json already exists in that project.
    $projectsRoot = Join-Path (Split-Path -Parent $TargetClaudeDir) 'Documents/GitHub'
    if (-not (Test-Path -LiteralPath $projectsRoot)) {
        $projectsRoot = (Resolve-Path -ErrorAction SilentlyContinue ~/Documents/GitHub).Path
    }
    if (-not $projectsRoot -or -not (Test-Path -LiteralPath $projectsRoot)) {
        Write-Note "skipping per-project local: no GitHub root found"
        return
    }
    $perProjectRoot = Join-Path $PayloadRoot 'templates/claude-code/per-project'
    $detectors = @(
        @{ Family = 'php';    Marker = 'composer.json' }
        @{ Family = 'cpp';    Marker = 'CMakeLists.txt' }
        @{ Family = 'python'; Marker = 'pyproject.toml' }
    )
    Get-ChildItem -LiteralPath $projectsRoot -Directory -ErrorAction SilentlyContinue | ForEach-Object {
        $proj = $_.FullName
        foreach ($det in $detectors) {
            $marker = Join-Path $proj $det.Marker
            if (-not (Test-Path -LiteralPath $marker -PathType Leaf)) { continue }
            $localDest = Join-Path $proj '.claude/settings.local.json'
            if (Test-Path -LiteralPath $localDest) {
                Write-Note "skip $proj (settings.local.json exists)"
                continue
            }
            $localSrc = Join-Path $perProjectRoot "$($det.Family)/.claude/settings.local.json"
            if (-not (Test-Path -LiteralPath $localSrc -PathType Leaf)) {
                Write-Warn2 "per-project template missing: $localSrc"
                continue
            }
            if ($DryRun) {
                Write-Note "would drop $($det.Family) local into $localDest"
            } else {
                New-Item -ItemType Directory -Force -Path (Split-Path -Parent $localDest) | Out-Null
                Copy-Item -LiteralPath $localSrc -Destination $localDest -Force
            }
            Write-Change "per-project [$($det.Family)] -> $localDest"
            break
        }
    }
}

# ---- Target installers -----------------------------------------------------

function Install-ClaudeTarget {
    $targetDir = Join-Path $HOME '.claude'
    if (-not (Test-Path -LiteralPath $targetDir)) {
        Write-Warn2 "Claude Code home missing ($targetDir). Creating."
        if (-not $DryRun) { New-Item -ItemType Directory -Force -Path $targetDir | Out-Null }
    }
    Write-Step "Installing Claude Code kit -> $targetDir"
    $backupDir = New-BackupFolder $targetDir

    Merge-SettingsJson `
        -TargetPath (Join-Path $targetDir 'settings.json') `
        -TemplatePath (Join-Path $PayloadRoot 'templates/claude-code/settings.json.template') `
        -BackupDir $backupDir `
        -TargetDir $targetDir

    $claudeMd = Join-Path $targetDir 'CLAUDE.md'
    Update-MarkdownSection `
        -TargetPath $claudeMd `
        -MarkerName 'wizard-kit:claude-md' `
        -NewContent (Get-Content -LiteralPath (Join-Path $PayloadRoot 'templates/claude-code/CLAUDE.md.template') -Raw -Encoding utf8) `
        -BackupDir $backupDir `
        -TargetDir $targetDir

    $memoryDir = Join-Path $targetDir 'memory'
    if (-not (Test-Path -LiteralPath $memoryDir)) {
        if (-not $DryRun) { New-Item -ItemType Directory -Force -Path $memoryDir | Out-Null }
    }
    $memoryMd = Join-Path $memoryDir 'MEMORY.md'
    if (-not (Test-Path -LiteralPath $memoryMd)) {
        if (-not $DryRun) {
            Copy-Item -LiteralPath (Join-Path $PayloadRoot 'templates/claude-code/MEMORY.md.template') -Destination $memoryMd -Force
        }
        Write-Change "create $memoryMd"
    } else {
        Backup-File -SourcePath $memoryMd -BackupDir $backupDir -TargetDir $targetDir
    }

    $planIndex = Build-PlanIndex (Join-Path $targetDir 'plans')
    if ([string]::IsNullOrWhiteSpace($planIndex)) {
        $planIndex = '<!-- (no plans found in ~/.claude/plans/) -->'
    }
    Update-IndexBlock -TargetPath $memoryMd -MarkerName 'wizard-kit:plan-index' -IndexBody $planIndex

    $skillsDest = Join-Path $targetDir 'skills'
    if (-not $DryRun) { New-Item -ItemType Directory -Force -Path $skillsDest | Out-Null }
    $installedSkills = Copy-Skills `
        -PayloadSkillRoot (Join-Path $PayloadRoot 'skills/claude-code') `
        -TargetSkillRoot $skillsDest `
        -BackupDir $backupDir `
        -TargetDir $targetDir

    Drop-PerProjectLocal $targetDir

    Update-ManifestEntry -TargetDir $targetDir -ManifestEntry @{
        version       = $KitVersion
        timestamp     = (Get-Date).ToString('o')
        action        = 'install'
        installer     = 'install-claude-kit.ps1'
        backup_dir    = ([IO.Path]::GetRelativePath($targetDir, $backupDir))
        skills_added  = @($installedSkills)
    }
}

function Install-CodexTarget {
    $targetDir = Join-Path $HOME '.codex'
    if (-not (Test-Path -LiteralPath $targetDir)) {
        Write-Warn2 "Codex home missing ($targetDir). Creating."
        if (-not $DryRun) { New-Item -ItemType Directory -Force -Path $targetDir | Out-Null }
    }
    Write-Step "Installing Codex kit -> $targetDir"
    $backupDir = New-BackupFolder $targetDir

    $configToml = Join-Path $targetDir 'config.toml'
    $tomlBody = Get-Content -LiteralPath (Join-Path $PayloadRoot 'templates/codex/config.toml.template') -Raw -Encoding utf8
    Update-TomlSection `
        -TargetPath $configToml `
        -NewBody $tomlBody `
        -BackupDir $backupDir `
        -TargetDir $targetDir

    $agentsMd = Join-Path $targetDir 'AGENTS.md'
    Update-MarkdownSection `
        -TargetPath $agentsMd `
        -MarkerName 'wizard-kit:agents-md' `
        -NewContent (Get-Content -LiteralPath (Join-Path $PayloadRoot 'templates/codex/AGENTS.md.template') -Raw -Encoding utf8) `
        -BackupDir $backupDir `
        -TargetDir $targetDir

    $sessionIndex = Build-SessionIndex (Join-Path $targetDir 'sessions')
    if ([string]::IsNullOrWhiteSpace($sessionIndex)) {
        $sessionIndex = '<!-- (no sessions found in ~/.codex/sessions/) -->'
    }
    Update-IndexBlock -TargetPath $agentsMd -MarkerName 'wizard-kit:session-index' -IndexBody $sessionIndex

    $skillsDest = Join-Path $targetDir 'skills'
    if (-not $DryRun) { New-Item -ItemType Directory -Force -Path $skillsDest | Out-Null }
    $installedSkills = Copy-Skills `
        -PayloadSkillRoot (Join-Path $PayloadRoot 'skills/codex') `
        -TargetSkillRoot $skillsDest `
        -BackupDir $backupDir `
        -TargetDir $targetDir

    Update-ManifestEntry -TargetDir $targetDir -ManifestEntry @{
        version       = $KitVersion
        timestamp     = (Get-Date).ToString('o')
        action        = 'install'
        installer     = 'install-claude-kit.ps1'
        backup_dir    = ([IO.Path]::GetRelativePath($targetDir, $backupDir))
        skills_added  = @($installedSkills)
    }
}

function Invoke-Rollback([string] $TargetDir) {
    if (-not (Test-Path -LiteralPath $TargetDir)) {
        Write-Warn2 "target missing: $TargetDir"
        return
    }
    Write-Step "Rolling back -> $TargetDir"
    $backupRoot = Join-Path $TargetDir '.wizard-kit-backup'
    if (-not (Test-Path -LiteralPath $backupRoot)) {
        Write-Warn2 "no backups to restore from: $backupRoot"
        return
    }
    $latest = Get-ChildItem -LiteralPath $backupRoot -Directory |
        Sort-Object Name -Descending | Select-Object -First 1
    if (-not $latest) {
        Write-Warn2 "no backup snapshots present"
        return
    }
    Write-Note "restoring from $($latest.FullName)"
    Get-ChildItem -LiteralPath $latest.FullName -File -Recurse | ForEach-Object {
        $rel = [IO.Path]::GetRelativePath($latest.FullName, $_.FullName)
        $dest = Join-Path $TargetDir $rel
        if ($DryRun) {
            Write-Note "would restore $rel"
        } else {
            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dest) | Out-Null
            Copy-Item -LiteralPath $_.FullName -Destination $dest -Force
        }
        Write-Change "restore $rel"
    }
    Update-ManifestEntry -TargetDir $TargetDir -ManifestEntry @{
        version    = $KitVersion
        timestamp  = (Get-Date).ToString('o')
        action     = 'rollback'
        installer  = 'install-claude-kit.ps1'
        from       = $latest.Name
    }
}

# ---- Main ------------------------------------------------------------------

Write-Host ''
Write-Step "Wizard Kit installer v$KitVersion"
Write-Note "payload: $PayloadRoot"
if ($DryRun)   { Write-Note 'mode: DRY-RUN (no writes)' }
elseif ($Rollback) { Write-Note 'mode: ROLLBACK' }
else           { Write-Note 'mode: INSTALL' }
Write-Host ''

if (-not $Force -and -not $DryRun) {
    $confirm = Read-Host 'Proceed? (y/N)'
    if ($confirm -notmatch '^[Yy]') {
        Write-Host 'Aborted.' -ForegroundColor Yellow
        exit 0
    }
}

if ($Rollback) {
    if (-not $CodexOnly)  { Invoke-Rollback (Join-Path $HOME '.claude') }
    if (-not $ClaudeOnly) { Invoke-Rollback (Join-Path $HOME '.codex') }
} else {
    if (-not $CodexOnly)  { Install-ClaudeTarget }
    if (-not $ClaudeOnly) { Install-CodexTarget }
}

Write-Host ''
Write-Host 'Done.' -ForegroundColor Green
