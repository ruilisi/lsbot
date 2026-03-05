# Auto-Approval Mode (`--yes`)

## Overview

The `--yes` (or `-y`) flag enables **auto-approval mode**, which disables all confirmation prompts and allows the AI to execute operations immediately without asking for permission.

This is useful when you trust the AI completely and want maximum automation without interruption.

## Quick Start

```bash
# Enable auto-approval with long flag
lingti-bot --yes router --provider deepseek --api-key sk-xxx

# Enable auto-approval with short flag
lingti-bot -y router --provider deepseek --api-key sk-xxx

# Combine with debug mode
lingti-bot --yes --debug router --provider deepseek --api-key sk-xxx
```

## What Changes with `--yes`?

### Without `--yes` (Default Behavior)

The AI may be cautious and ask for confirmation before executing sensitive operations:

**Example conversation:**
```
User: Save this content to README_EN.md

AI: ❌ I haven't actually saved the file yet.
    All operations were read-only (analysis and translation).

    Would you like me to save the updated README_EN.md now?
    Please confirm by saying "Yes, save the file" or "Proceed with save"
```

### With `--yes` (Auto-Approval Mode)

The AI immediately executes the requested operation without asking:

**Example conversation:**
```
User: Save this content to README_EN.md

AI: ✅ File saved successfully to README_EN.md (2,847 bytes written)
```

## Behavior Changes

When `--yes` is enabled, the AI receives these instructions:

1. **Execute file writes immediately** - No confirmation needed for creating/updating files
2. **Run shell commands directly** - No permission prompts for command execution
3. **Delete/modify files without hesitation** - Trust user intent completely
4. **Skip safety prompts** - User has explicitly disabled all warnings
5. **Only reject impossible/dangerous operations** - Like `rm -rf /` or system-breaking commands

## Use Cases

### ✅ When to Use `--yes`

- **Batch operations**: Processing multiple files or tasks automatically
- **Trusted environment**: Running in your own development environment
- **Known safe operations**: File edits, documentation updates, code generation
- **CI/CD pipelines**: Automated workflows where manual approval isn't feasible
- **Power users**: You understand the risks and want maximum efficiency

### ❌ When NOT to Use `--yes`

- **Production servers**: Direct operations on live systems
- **Shared environments**: Multi-user systems where mistakes affect others
- **Learning/testing**: First time using a feature or command
- **Uncertain operations**: When you're not sure what will happen
- **Sensitive data**: Operations involving credentials, keys, or private information

## Safety Considerations

### What `--yes` Does NOT Disable

Even with auto-approval enabled, the AI will still refuse:

1. **Destructive system commands**: `rm -rf /`, `mkfs`, `dd if=/dev/zero`
2. **Dangerous operations**: Deleting critical system files or directories
3. **Impossible operations**: Tasks that would break the system
4. **Out-of-scope actions**: Operations outside the AI's capabilities

### What You Should Still Monitor

- File system changes (use `git status` to track)
- Shell command outputs (check logs with `--debug`)
- Network operations (external API calls, downloads)
- Resource usage (CPU, memory, disk space)

## Examples

### Example 1: Documentation Update

**Without `--yes`:**
```bash
$ lingti-bot gateway --provider qwen --api-key sk-xxx

User: Update the README with the new feature description

AI: I've prepared the updated content. Should I write it to README.md?
    Please confirm.

User: Yes, please save it

AI: ✅ File updated successfully
```

**With `--yes`:**
```bash
$ lingti-bot --yes router --provider qwen --api-key sk-xxx

User: Update the README with the new feature description

AI: ✅ README.md updated with new feature description (3 sections added)
```

### Example 2: Batch File Operations

**Without `--yes`:**
```bash
User: Rename all .txt files to .md in the docs folder

AI: I found 15 .txt files. Should I proceed with renaming them all?

User: Yes

AI: ✅ Renamed 15 files
```

**With `--yes`:**
```bash
User: Rename all .txt files to .md in the docs folder

AI: ✅ Renamed 15 files in docs/ (.txt → .md)
```

### Example 3: Code Generation

**Without `--yes`:**
```bash
User: Create a new API endpoint for user authentication

AI: I've generated the code for the endpoint. Should I create the new file
    at src/api/auth.go?

User: Go ahead

AI: ✅ Created src/api/auth.go
```

**With `--yes`:**
```bash
User: Create a new API endpoint for user authentication

AI: ✅ Created src/api/auth.go with authentication endpoint
    - POST /api/auth/login
    - POST /api/auth/logout
    - Middleware for token validation
```

## Command Reference

### Global Flag

```bash
lingti-bot [--yes|-y] <command> [options]
```

- **Flag**: `--yes` or `-y`
- **Scope**: Global (works with all commands)
- **Default**: `false` (auto-approval disabled)
- **Type**: Boolean (no value needed)

### Combining with Other Flags

```bash
# Auto-approval + Debug mode
lingti-bot --yes --debug router --provider deepseek --api-key sk-xxx

# Auto-approval + Verbose logging
lingti-bot -y --log verbose router --provider deepseek --api-key sk-xxx

# Auto-approval + Custom debug directory
lingti-bot --yes --debug --debug-dir /tmp/my-debug router [...]
```

## Environment Variables

Currently, auto-approval can **only** be enabled via command-line flag. There is no environment variable equivalent.

If you need persistent auto-approval, consider:

1. Creating a shell alias:
   ```bash
   alias lingti='lingti-bot --yes'
   ```

2. Using a wrapper script:
   ```bash
   #!/bin/bash
   lingti-bot --yes "$@"
   ```

## Troubleshooting

### AI Still Asks for Confirmation

**Possible causes:**
1. Flag placed incorrectly (must be before subcommand):
   ```bash
   # ❌ Wrong
   lingti-bot gateway --yes --provider deepseek --api-key sk-xxx

   # ✅ Correct
   lingti-bot --yes router --provider deepseek --api-key sk-xxx
   ```

2. Using old binary (rebuild after updating):
   ```bash
   go build -o dist/lingti-bot .
   ```

### How to Verify Auto-Approval is Enabled

Check the logs at startup:
```bash
lingti-bot --yes --log verbose router [...]
```

In verbose mode, you should see the system prompt includes:
```
## 🚀 AUTO-APPROVAL MODE ENABLED
```

## Best Practices

1. **Version Control**: Always use `--yes` in git-tracked directories
   - Easy to review changes: `git diff`
   - Easy to undo mistakes: `git reset --hard`

2. **Start with `--debug`**: First time using auto-approval?
   ```bash
   lingti-bot --yes --debug router [...]
   ```
   This shows all operations being performed.

3. **Test in Safe Directory**: Try auto-approval in a test folder first
   ```bash
   mkdir /tmp/test-auto-approve
   cd /tmp/test-auto-approve
   lingti-bot --yes router [...]
   ```

4. **Review Regularly**: Periodically check what was changed
   ```bash
   git log --oneline --name-only
   git diff HEAD~5
   ```

5. **Use with Specific Tasks**: Enable only for specific operations
   ```bash
   # For file operations
   lingti-bot --yes router --provider deepseek --api-key sk-xxx

   # Regular mode for exploratory tasks
   lingti-bot gateway --provider deepseek --api-key sk-xxx
   ```

## Security Notes

- **Not a Security Feature**: `--yes` is about convenience, not security
- **Trust Required**: Only use when you trust the AI's judgment
- **Audit Trail**: Always keep logs (`--log verbose`) when using auto-approval
- **Backup First**: Have backups before enabling auto-approval on important files
- **Gradual Adoption**: Start with read-only tasks, then gradually enable for writes

## Comparison Table

| Feature | Without `--yes` | With `--yes` |
|---------|-----------------|--------------|
| File writes | May ask for confirmation | Executes immediately |
| File deletions | May ask for confirmation | Executes immediately |
| Shell commands | May ask for confirmation | Executes immediately |
| Dangerous operations | Blocked + warning | Blocked + warning |
| Read operations | Always allowed | Always allowed |
| User experience | Safer, more prompts | Faster, fewer prompts |
| Best for | Exploratory tasks, learning | Automation, batch processing |

## Related Flags

- `--debug`: Enable debug mode (shows all operations)
- `--log <level>`: Set logging level (silent, info, verbose, very-verbose)
- `--debug-dir <path>`: Set debug screenshot directory

## FAQ

**Q: Can I set auto-approval as default?**
A: Currently no. You must explicitly pass `--yes` each time. This is by design for safety.

**Q: Does `--yes` bypass all security checks?**
A: No. It only disables confirmation prompts. Dangerous operations (like `rm -rf /`) are still blocked.

**Q: Can I use `--yes` with the MCP server mode?**
A: The `--yes` flag currently only affects the `router` command. Other commands may not support it.

**Q: What if I want to cancel after using `--yes`?**
A: Use Ctrl+C to stop the program at any time. This works with or without `--yes`.

**Q: Does `--yes` make operations faster?**
A: No. It only removes confirmation prompts. The actual operations take the same time.

## See Also

- [Browser Debug Mode](browser-debug.md) - Debug browser automation
- [Global Flags](../README.md#global-flags) - All available global flags
- [Security Configuration](../README.md#security) - Configure security settings
