# helmfmt

`helmfmt` is a small CLI to auto-align indentation in Helm templates. It walks chart templates recursively and normalizes indentation for control blocks, variables, and simple functions, while respecting comments.

It can be configured for [Zed IDE](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#zed-ide-configuration)/[VS Code](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#vscode-ide-configuration), and as a [pre-commit hook](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#pre-commit-hook-configuration).

---

## What gets formatted

Only lines starting with Go-template tags. YAML indentation is untouched.

These are indented:

- Control blocks: `range`, `with`, `define`, `block`
- Branching: `if`, `else`, `else if`, `end`
- Vars: `{{ $var := ... }}`
- Some functions: `include`, `fail`, `printf` etc.
- Comments: `{{/* ... */}}`

These are not indented:

- `tpl` and `toYaml` because they can break YAML indentation

---

## Example

**Before**

```helm
{{- if .Values.createNamespace }}
{{- range .Values.namespaces }}
apiVersion: v1
kind: Namespace
metadata:
  name: {{ . }}
{{- with $.Values.namespaceLabels }}
  labels:
{{ toYaml . | indent 4 }}
{{- end }}
---
{{- end }}
{{- end }}
```

**After**

```helm
{{- if .Values.createNamespace }}
  {{- range .Values.namespaces }}
apiVersion: v1
kind: Namespace
metadata:
  name: {{ . }}
    {{- with $.Values.namespaceLabels }}
  labels:
{{ toYaml . | indent 4 }}
    {{- end }}
---
  {{- end }}
{{- end }}
```

## Installation

### First method

```bash
go install github.com/digitalstudium/helmfmt@latest
```

Then you should add `$HOME/go/bin/` to PATH if not already done:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc && source ~/.bashrc
```

### Second method

```bash
git clone https://github.com/digitalstudium/helmfmt
cd helmfmt
go build
```

If you compiled it via `go build` then you can run it with:

```bash
./helmfmt <chart-path>
```

or add it to PATH via:

```bash
sudo install ./helmfmt /usr/local/bin/
```

### Third method

Download it from [releases](https://github.com/digitalstudium/helmfmt/releases) (for Linux only)

---

## Usage

```bash
helmfmt <chart-path>
helmfmt --files <file1> <file2> ...
```

Example run:

```bash
helmfmt ./mychart
[UPDATED] mychart/templates/deployment.yaml
Processed: 3, Updated: 1, Errors: 0
```

or

```bash
helmfmt --files ./mychart/templates/deployment.yaml ./mychart/templates/svc.yaml
[UPDATED] mychart/templates/deployment.yaml
Processed: 2, Updated: 1, Errors: 0
```

---

## pre-commit hook configuration

To use `helmfmt` as a pre-commit hook, add the following to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/digitalstudium/helmfmt
    rev: v0.1.0
    hooks:
      - id: helmfmt
```

## Zed IDE configuration

Add these lines to your `settings.json`:

```json
  "languages": {
    "Helm": {
      "formatter": {
        "external": {
          "command": "helmfmt",
          "arguments": ["--files", "{buffer_path}", "--stdout"]
        }
      }
    }
  }
```

In addition, you should install helm language extension https://github.com/cabrinha/helm.zed
and helm language server https://github.com/mrjosh/helm-ls

## VSCode IDE configuration

Add these lines to your settings:

```json
  "advancedLocalFormatters.formatters": [
    {
      "command": ["helmfmt", "--files", "$absoluteFilePath", "--stdout"],
      "languages": ["helm"]
    }
  ],
  "[helm]": {
    "editor.defaultFormatter": "webfreak.advanced-local-formatters"
  }
```

In addition these extensions should be installed:
https://github.com/WebFreak001/vscode-advanced-local-formatters
https://github.com/vscode-kubernetes-tools/vscode-kubernetes-tools

---

## Roadmap

- Check-only / diff mode
- More Helm funcs (dict, etc.)
- Format spaces inside tags

---

**Issues and PRs welcome!**
