# helmfmt

`helmfmt` is a small CLI to auto-align indentation in Helm templates. It walks chart templates recursively and normalizes indentation for control blocks, variables, and simple functions, while respecting comments.

- Input: `.yaml`, `.yml`, `.tpl` in `templates` folder of chart
- Output: rewritten only if indentation changed
- Indent: 2 spaces

> [!NOTE]  
> This tool is under development.
> I tested it on the multiple big charts, and it didn't break anything.
> Example of charts which I used for testing:
>
> https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack (175 templates)
> 
> https://github.com/VictoriaMetrics/helm-charts/tree/master/charts/victoria-logs-cluster (14 templates)
> 
> But I don't guarantee that it would work for all cases.

---

## Example

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

## What gets formatted

Only lines with Go-template tags. YAML indentation is untouched.

Supported:

- Control blocks: `if`, `range`, `with`, `define`, `block`
- Branching: `else`, `else if`, `end`
- Vars: `{{ $var := ... }}`
- Simple functions: `include`, `fail`, `printf` etc.
- Block comments: `{{/* ... */}}`

---

## Roadmap

- Check-only / diff mode
- More Helm funcs (tpl, dict, etc.)
- Golden tests/examples
- Optional YAML alignment around tags

---

**Issues and PRs welcome!**
