# helmfmt

`helmfmt` is a small CLI to auto-align indentation in Helm templates. It walks chart templates recursively and normalizes indentation for control blocks, variables, and simple functions, while respecting comments.

- Input: `.yaml`, `.yml`, `.tpl` in `templates` folder of chart
- Output: rewritten only if indentation changed
- Indent: 2 spaces

---

## Install

- Requires Go 1.22+
- Run:
  ```bash
  go install github.com/digitalstudium/helmfmt@latest
  ```

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

## What gets formatted

Only lines with Go-template tags. YAML indentation is untouched.

Supported:

- Control blocks: `if`, `range`, `with`, `define`, `block`
- Branching: `else`, `else if`, `end`
- Vars: `{{ $var := ... }}`
- Simple functions: `include`, `fail`, `printf` etc.
- Block comments: `{{/* ... */}}`

---

## Examples

**Before**

```gotmpl
{{ if .Values.enabled }}
{{ if .Values.debug }}
{{ $replicas := .Values.replicas }}
{{ include "chart.labels" . }}
{{ end }}
{{ end }}
```

**After**

```gotmpl
{{ if .Values.enabled }}
  {{ if .Values.debug }}
    {{ $replicas := .Values.replicas }}
    {{ include "chart.labels" . }}
  {{ end }}
{{ end }}
```

---

## Roadmap

- Check-only / diff mode
- More Helm funcs (tpl, dict, etc.)
- Golden tests/examples
- Optional YAML alignment around tags

---

## Build from source

```bash
git clone https://github.com/digitalstudium/helmfmt
cd helmfmt
go build -o helmfmt
./helmfmt <chart-path>
```

---

**Issues and PRs welcome!**
