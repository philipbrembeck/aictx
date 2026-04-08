# Changelog

## v0.1.0 (2026-04-08)

### Breaking changes

**Config schema: per-context provider model**

`Provider`, `Options`, and `HasKeyringKey` have moved from per-`TargetEntry` to the `Context` level. All targets within a context now share one API key and model.

Old format (per-target):
```yaml
contexts:
  - name: work
    targets:
      - id: claude-code-cli
        provider:
          endpoint: https://api.example.com
          model: claude-opus-4-6
        options:
          alwaysThinking: true
        hasKeyringKey: true
```

New format (per-context):
```yaml
contexts:
  - name: work
    provider:
      endpoint: https://api.example.com
      model: claude-opus-4-6
    options:
      alwaysThinking: true
    hasKeyringKey: true
    targets:
      - id: claude-code-cli
      - id: claude-code-vscode
```

**Keyring account key changed**

Old: `aictx / contextName/targetID`
New: `aictx / contextName`

### Migration

Migration is **automatic and transparent**. On first run after upgrading, `aictx` detects the old schema and migrates in place:

1. Lifts the first non-empty `Provider` found across targets to the context level
2. Lifts `Options` to the context level
3. Migrates keyring entries from `contextName/targetID` to `contextName`
4. Saves the updated config

If targets had different API keys, a warning is printed to stderr and the first non-empty key is kept. Re-run `aictx copy <ctx> <ctx> --api-key <new-key>` to update.

#### Back up your API keys before upgrading

The migration moves keyring entries automatically, but if something goes wrong you want your keys handy. Run the script for your platform before upgrading and save the output somewhere safe (local file, password manager note, etc.).

**macOS** — uses the `security` CLI (built-in):

```sh
#!/usr/bin/env sh
# Reads context names from config.yaml and dumps each keyring entry.
CONFIG="$HOME/.config/aictx/config.yaml"
echo "=== aictx keyring backup (macOS) ==="
grep "^  - name:" "$CONFIG" | sed 's/.*name: //' | while read -r ctx; do
  # Old format: contextName/targetID
  for tid in claude-code-cli claude-code-vscode pi-cli; do
    key=$(security find-generic-password -s aictx -a "$ctx/$tid" -w 2>/dev/null)
    [ -n "$key" ] && echo "account=$ctx/$tid  key=$key"
  done
  # New format: contextName (already migrated)
  key=$(security find-generic-password -s aictx -a "$ctx" -w 2>/dev/null)
  [ -n "$key" ] && echo "account=$ctx  key=$key"
done
```

**Linux** — uses `secret-tool` (install via `apt install libsecret-tools` / `dnf install libsecret`):

```sh
#!/usr/bin/env sh
CONFIG="$HOME/.config/aictx/config.yaml"
echo "=== aictx keyring backup (Linux) ==="
grep "^  - name:" "$CONFIG" | sed 's/.*name: //' | while read -r ctx; do
  for tid in claude-code-cli claude-code-vscode pi-cli; do
    key=$(secret-tool lookup service aictx account "$ctx/$tid" 2>/dev/null)
    [ -n "$key" ] && echo "account=$ctx/$tid  key=$key"
  done
  key=$(secret-tool lookup service aictx account "$ctx" 2>/dev/null)
  [ -n "$key" ] && echo "account=$ctx  key=$key"
done
```

**Windows** — uses PowerShell and Windows Credential Manager:

```powershell
# Run in PowerShell. Reads config.yaml and dumps each Credential Manager entry.
$config = "$env:APPDATA\..\Local\aictx\config.yaml"
if (-not (Test-Path $config)) { $config = "$env:HOME\.config\aictx\config.yaml" }

Write-Host "=== aictx keyring backup (Windows) ==="
$contexts = (Get-Content $config | Select-String "^  - name:").Matches.Value -replace ".*name: "

Add-Type -AssemblyName System.Runtime.WindowsRuntime
[Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime] | Out-Null
$vault = New-Object Windows.Security.Credentials.PasswordVault

foreach ($ctx in $contexts) {
    foreach ($account in @("$ctx/claude-code-cli", "$ctx/claude-code-vscode", "$ctx/pi-cli", $ctx)) {
        try {
            $cred = $vault.Retrieve("aictx", $account)
            $cred.RetrievePassword()
            Write-Host "account=$account  key=$($cred.Password)"
        } catch {}
    }
}
```

### New features

- **`aictx targets [contextname]`** — Checkbox multi-select picker to add or remove targets from a context. Pre-checks currently configured targets; detected targets are labelled. Falls back to a plain list when stdout is not a terminal.
- **`aictx version`** — Prints `aictx v0.1.0`.
- **pi CLI target** (`pi-cli`) — Support for the pi Coding Agent CLI.
- **`picker.PickMulti`** — Internal checkbox picker reused by `targets` and `add`.

### Changes

- `aictx add` interactive mode now uses the checkbox picker for target selection (detected targets pre-checked). Provider and Options are prompted once at the context level.
- `aictx copy` `--target` flag now only scopes `--env` overrides; provider/options flags always apply at the context level.
- `aictx rename` and `aictx rm` now operate on a single context-level keyring entry.

### Notes for users who script against the config YAML

The `targets[].provider` and `targets[].options` fields are removed. Scripts reading or writing `config.yaml` directly must be updated to use the top-level `provider` and `options` fields on each context.
