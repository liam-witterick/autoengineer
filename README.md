# AutoEngineer 🤖

**Your repositories improve themselves.**

AutoEngineer continuously discovers infrastructure, security, and pipeline issues in your codebase — then creates GitHub Issues and delegates fixes to Copilot. 

No backlog grooming. No manual audits. No "we'll get to it later."

> Copilot finds the work.  Copilot does the work. AutoEngineer connects the two. 

---

## The Problem

AI coding assistants are powerful — but they only fix what you ask them to. 

| AI Assistants Today | The Gap |
|---------------------|---------|
| Wait for you to prompt them | Who finds the problems? |
| Fix what you tell them to | Who writes the tickets? |
| One-shot interactions | Who connects discovery to resolution? |

**You're still the bottleneck.** You have to notice the issue, write it up, prompt the AI, and link the PR back. That's a lot of glue work.

## The Solution

AutoEngineer lets Copilot find its own work:

```
🔍 DISCOVER  →  Copilot CLI scans your repo, finds issues
     ↓
📋 TRACK     →  GitHub Issues created automatically
     ↓
🔧 FIX       →  Copilot CLI (local) or Coding Agent (cloud) resolves them
     ↓
👀 REVIEW    →  You review and merge the PR
     ↓
🔗 CLOSE     →  Issue closed, changes merged
     ↓
     └──────────────── repeat ────────────────┘
```

**Work that would never have been created now exists — and gets resolved.** AutoEngineer automates discovery and fix delegation, but you stay in control by reviewing and merging PRs.

---

## What It Finds

AutoEngineer discovers improvements across your DevOps stack:

### 🔒 Security
- Over-permissive IAM roles and RBAC
- Open security groups (0.0.0.0/0)
- Hardcoded secrets and credentials
- Missing encryption, weak TLS configs
- Containers running as root

### 🏗️ Infrastructure
- Unpinned Terraform/OpenTofu module versions
- Missing resource tags and naming inconsistencies
- Kubernetes manifests lacking resource limits
- Helm chart misconfigurations
- Cost optimization opportunities

### ⚙️ Pipelines
- Deprecated GitHub Actions
- Missing cache configurations
- Inefficient workflow triggers
- Duplicated workflow logic
- Slow builds that could be parallelized

---

## External Scanner Integration

AutoEngineer automatically integrates with popular security scanners when installed, running them in parallel with Copilot analysis for comprehensive coverage.

### Default Scanners (Auto-Detected)

| Scanner | What It Finds | Auto-Run |
|---------|---------------|----------|
| **Checkov** | IaC security, compliance policies | ✅ When installed |
| **Trivy** | Misconfigurations, vulnerabilities | ✅ When installed |

**No configuration needed** — AutoEngineer detects installed scanners and runs them automatically. Findings from all sources are merged and deduplicated.

### Quick Start

```bash
# Install scanners (optional, but recommended)
pip install checkov
brew install trivy  # or: curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh

# Run with scanners (automatic)
autoengineer

# Skip scanners for faster runs
autoengineer --no-scanners
# or
autoengineer --fast

# Check which scanners are detected
autoengineer --check
```

### Scanner Status Reporting

```bash
$ autoengineer --check

🔍 Checking dependencies...

   ✅ copilot (GitHub Copilot CLI 1.x.x)
   ✅ gh (GitHub CLI 2.x.x)
   ✅ gh authenticated

🔍 External Scanners:
   ✅ checkov (3.2.x)
   ✅ trivy (0.55.x)

✅ All dependencies are installed
```

### Configuration

Create `.github/autoengineer.yaml` for advanced scanner control:

```yaml
scanners:
  # Disable specific scanners
  disabled:
    - checkov  # Skip Checkov even if installed
  
  # Enable cloud scanners (requires API keys)
  enabled:
    - aikido
  
  # Cloud scanner config
  aikido:
    api_key_env: "AIKIDO_API_KEY"
```

### How It Works

1. **Auto-Detection**: On each run, AutoEngineer checks for installed scanners
2. **Parallel Execution**: Scanners run concurrently with Copilot analysis for speed
3. **Deduplication**: Findings are merged and similar issues removed automatically
4. **Silent Fallback**: Missing scanners are skipped without errors

**Output Example:**

```bash
$ autoengineer

🔍 Running analysis...
   ✅ checkov: 12 finding(s)
   ✅ trivy: 8 finding(s)
   ✅ copilot: 5 finding(s)
   
📊 Scanner Summary:
   ✅ checkov: ran successfully
   ✅ trivy: ran successfully
   
📋 NEW FINDINGS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Summary: 🔴 High: 3  🟡 Medium: 15  🟢 Low: 7  (Total: 25)
```

---

## Quick Start

### Install

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/liam-witterick/autoengineer/master/install.sh | bash

# Or download from releases
# https://github.com/liam-witterick/autoengineer/releases
```

### Requirements

| Dependency | Why | Install |
|------------|-----|---------|
| GitHub Copilot | Does the analysis and fixing | [Copilot subscription](https://github.com/features/copilot) |
| `gh` CLI | GitHub API access | [Install](https://cli.github.com/) |

```bash
# Verify setup
autoengineer --check
```

### Run

```bash
cd /path/to/your/repo

# Interactive mode — review findings, choose what to action
autoengineer

# Focus on one area
autoengineer --scope security
autoengineer --scope pipeline
autoengineer --scope infra

# Use custom instructions to guide analysis
autoengineer --instructions ./my-custom-instructions.md
autoengineer --instructions-text "Focus on Terraform security issues"

# Full automation — find issues, create tickets, delegate fixes
# (PRs are created automatically, you review and merge them)
autoengineer --create-issues --delegate
```

---

## How It Works

### Interactive Mode (default)

```bash
$ autoengineer

🤖 AUTOENGINEER
===============

🔍 Running scoped analyses...

   ✅ security: 3 finding(s)
   ✅ pipeline: 2 finding(s)
   ✅ infra: 4 finding(s)

📊 Results: 🔴 High: 2  🟡 Medium: 4  🟢 Low: 3

━━ 🔒 Security ━━

1. 🔴 Security group allows ingress from 0.0.0.0/0
      Files: infra/security.tf

2. 🟡 IAM role has wildcard permissions
      Files: infra/iam.tf

...

Action: [f]ix, [l]ater, [p]review, [q]uit:
```

**Interactive Menu Options:**

- **`[f]ix`** - Shows both existing tracked issues AND new findings in one unified list. Select what to fix, then choose:
  - **`[l]ocal`** - Fix with Copilot CLI (immediate, local changes)
  - **`[c]loud`** - Create issue (if needed) + delegate to Copilot coding agent (automated PR)
- **`[l]ater`** - Create GitHub issues for new findings to track them without fixing now
- **`[p]review`** - Show findings summary again
- **`[q]uit`** - Exit (findings are saved to `findings.json`)

**Delegation Protection:** Issues are automatically labeled as `delegated` to prevent double-delegation.

### Automated Mode

```bash
# Create issues for all findings
autoengineer --create-issues

# Create issues AND delegate fixes to Copilot (creates PRs for you to review)
autoengineer --create-issues --delegate

# Only action high-severity findings
autoengineer --create-issues --delegate --min-severity high
```

**Note:** When using `--delegate`, AutoEngineer creates PRs automatically, but you still need to review and merge them. Nothing changes in your repository without your approval.

---

## Configuration

### Customize What Gets Flagged

Create `.github/copilot-instructions.md`:

```markdown
## High Priority
- Flag any security groups open to 0.0.0.0/0
- Check for hardcoded secrets
- Ensure all resources have required tags

## Ignore
- Don't flag test fixtures
- Skip example directories
```

**Note:** AutoEngineer automatically loads this file if it exists. You can also:
- Use `--instructions <path>` to specify a different instructions file
- Use `--instructions-text "your instructions"` to pass instructions directly on the command line

### Ignore Specific Findings

Create `.github/autoengineer-ignore.yaml`:

```yaml
# Accepted risks
accepted:
  - title: "Security issue in production environment"
    reason: "Legacy system, decommissioning Q2"
    accepted_by: "security-team"

# Paths to skip
ignore_paths:
  - "examples/*"
  - "test/fixtures/*"

# Pattern matching (case-insensitive)
ignore_patterns:
  - "*sandbox*"
  - "*demo*"
```

---

## CLI Reference

| Flag | Description |
|------|-------------|
| `--scope <type>` | Focus analysis: `security`, `pipeline`, `infra`, or `all` (default) |
| `--create-issues` | Automatically create GitHub Issues for findings |
| `--delegate` | Delegate fixes to Copilot Coding Agent (requires `--create-issues`) |
| `--min-severity <level>` | Only action findings at this level or above: `low`, `medium`, `high` |
| `--output <path>` | Save findings to specified file (default: `./findings.json`) |
| `--use-existing-findings` | Load findings from file instead of running a new scan |
| `--instructions <path>` | Path to custom instructions file (overrides `.github/copilot-instructions.md`) |
| `--instructions-text <text>` | Custom instructions as text (overrides file-based instructions) |
| `--no-scanners` | Skip external scanner integration |
| `--fast` | Fast mode - skip scanners (alias for `--no-scanners`) |
| `--check` | Verify dependencies and show scanner status |

### Reusing Findings

Save time by reusing previously saved findings instead of running a new scan:

```bash
# Run scan and save findings
autoengineer --output ./my-scans/results.json

# Later, resume with those findings
autoengineer --use-existing-findings --output ./my-scans/results.json

# Or use default location
autoengineer --use-existing-findings  # loads from findings.json

# Combine with other flags
autoengineer --use-existing-findings --min-severity high
autoengineer --use-existing-findings --create-issues
```

**Note:** When using `--use-existing-findings`, AutoEngineer still fetches existing tracked issues from GitHub to show both saved findings and tracked issues in the session.

---

## FAQ

**Does this modify my code?**
No. AutoEngineer creates issues and can delegate fixes to Copilot, which creates PRs. However, nothing changes in your repository without your explicit review and approval — you must review and merge all PRs yourself.

**What if I disagree with a finding?**
Close the issue, or add it to your ignore config. AutoEngineer won't flag it again.

**Does this work with private repos?**
Yes — as long as you have Copilot access and `gh` is authenticated.

**How is this different from linters/scanners?**
Linters find problems. AutoEngineer finds problems AND creates the tickets AND delegates the fixes. It's the full loop. Plus, AutoEngineer integrates with external scanners (Checkov, Trivy) automatically, combining their findings with Copilot's analysis for comprehensive coverage.

---

## Roadmap

- [ ] CloudFormation, Pulumi, CDK support
- [ ] Custom severity rules
- [ ] Slack/Teams notifications
- [ ] Metrics dashboard
- [ ] Interactive TUI mode

---

## License

MIT

---

**Built with ❤️ by [liam-witterick](https://github.com/liam-witterick)**
