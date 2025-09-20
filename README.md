# helmfmt

`helmfmt` is a small CLI to auto-align indentation in Helm templates. It walks chart templates recursively and normalizes indentation for control blocks, variables, and simple functions, while respecting comments.

---

## What gets formatted

Only lines starting with Go-template tags. YAML indentation is untouched.

Supported:

- Control blocks: `range`, `with`, `define`, `block`
- Branching: `if`, `else`, `else if`, `end`
- Vars: `{{ $var := ... }}`
- Some functions: `include`, `fail`, `printf` etc.
- Comments: `{{/* ... */}}`

Not supported:

- `tpl` and `toYaml` because it can break YAML indentation

---

## Example

**Before**

```gotmpl
{{- if .Values.enabled }}
{{ range $foobar := .Values.list }}
{{/*
This is
a multiline comment
*/}}
{{- $var := .Values.someValue }}
{{- end }}
{{- end }}
```

**After**

```gotmpl
{{- if .Values.enabled }}
  {{ range $foobar := .Values.list }}
    {{/*
    This is
    a multiline comment
    */}}
    {{- $var := .Values.someValue }}
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
```

Example run:

```
helmfmt ./mychart
[UPDATED] mychart/templates/deployment.yaml
Processed: 3, Updated: 1, Errors: 0
```

---

## pre-commit hook configuration

To use `helmfmt` as a pre-commit hook, add the following to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/digitalstudium/helmfmt
    rev: v0.0.3
    hooks:
      - id: helmfmt
```

---

## Roadmap

- Check-only / diff mode
- More Helm funcs (dict, etc.)
- Format spaces inside tags
- Create Zed/VSCode extension

---

**Issues and PRs welcome!**
