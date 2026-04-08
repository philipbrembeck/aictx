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

The migration moves keyring entries automatically, but if something goes wrong you want your keys handy. Run this before upgrading:

```sh
#!/usr/bin/env sh
# Dumps all aictx keyring entries to stdout so you can save them somewhere safe.
# Requires the `security` CLI (macOS) or equivalent for your platform.

echo "=== aictx keyring backup ==="
# List all entries for the aictx service and print account + password
security find-generic-password -s aictx 2>/dev/null | grep "acct" | while read -r line; do
  account=$(echo "$line" | sed 's/.*"acct"<blob>="\(.*\)"/\1/')
  password=$(security find-generic-password -s aictx -a "$account" -w 2>/dev/null)
  echo "account=$account  key=$password"
done
```

Save the output somewhere safe (e.g. a local file or password manager note). After migration you can verify the new entries with:

```sh
security find-generic-password -s aictx -w  # single context-level entry
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
