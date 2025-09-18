# helmfmt

A small CLI that auto-aligns indentation in Helm templates. It walks chart/templates recursively and neatly aligns indentation for control blocks, variables, and “simple” functions (include, printf, etc.), while respecting block comments and inline constructs.

- Input: .yaml, .yml, .tpl files under a chart’s templates directory
- Output: files are rewritten only if indentation changed
- Indentation step: 2 spaces

## Installation

- Requires Go 1.22+
- Install:
  ```
  go install github.com/digitalstudium/helmfmt@latest
  ```

## Usage

```
helmfmt <chart-path>
```

Example:

```
helmfmt ./mychart
[UPDATED] mychart/templates/deployment.yaml
Processed: 3 files, Updated: 1, Errors: 0
```

- Wrong args: prints Usage: helmfmt <chart-path>
- If chart/templates is missing: prints an error and exits
- Otherwise, walks files recursively, reports updated files, and prints a summary

## What gets formatted

Only the indentation of lines that contain Go-template tags is changed. Plain YAML lines are left as-is.

Supported constructs:

- Control blocks: if, range, with, define, block
- Branching: else, else if
- Block end: end
- Variable declarations: lines starting with {{ $var ... }}
- Simple functions: include, fail, toYaml, printf
- Block comments: {{/* ... */}} (possibly multi-line)

## Parsing rules (from cases.md)

Tag boundaries:

- Tag open: line starts with {{ or {{- (after optional spaces)
- Tag close: }} or -}}
- Comment open: {{/\* (also supports {{- and spaces)
- Comment close: \*/}} (on the same or following lines)

Token detection at line start (skipping a leading comment if present):

- Variable declaration open:
  - {{ (0+ spaces) $var
  - {{- (1+ spaces) $var
  - Must close with }} or -}} on the same line
- Simple functions (include/fail/toYaml/printf) open:
  - Same as variables; must close on the same line
- Control start (if/range/with/define/block) open:
  - Same start forms as above
  - The closing }} or -}} can be on the same or following lines
  - If a comment opens inside the tag, the parser finds the comment’s end first, then continues searching for the tag close
- Control end (end) open:
  - Same start forms as above; must close on the same line

Handled scenarios:

- Line starts with a control start
- Line starts with a comment; the control start comes after the comment (same or next line)
- Line starts with a control end
- Line starts with a comment; the control end comes after the comment (same or next line)
- Line starts with a variable declaration
- Line starts with a comment; the variable declaration comes after the comment (same or next line)

## Indentation rules

- Control blocks increase indentation depth for subsequent content
- else / else if / end are dedented by one level relative to the current depth
- Variable declarations and simple functions are indented at the current depth
- Inline constructs (start and end on the same line) do not increase depth; they are indented like variables
- If a line starts with a comment and the actual token begins after the comment (same or next line), indentation is based on that token
- Multi-line tags (when the tag’s closing braces are on later lines) are indented consistently across all lines of the tag
- Comments inside tags are ignored correctly when searching for tag boundaries

Exception:

- If a control construct has its end on the same line as its start, those lines do not add indentation depth (same level as variables). There may be a comment between start and end.

## Examples

- Nested controls, variables, and include:
  Before:

  ```
  {{ if .Values.enabled }}
  {{ if .Values.debug }}
  {{ $replicas := .Values.replicas }}
  {{ include "chart.labels" . }}
  {{end}}
  {{end}}
  ```

  After:

  ```
  {{ if .Values.enabled }}
    {{ if .Values.debug }}
      {{ $replicas := .Values.replicas }}
      {{ include "chart.labels" . }}
    {{ end }}
  {{ end }}
  ```

- else and else if:
  Before:

  ```
  {{ if .Values.a }}
  {{ $x := 1 }}
  {{else if .Values.b}}
  {{ include "x" . }}
  {{else}}
  {{ include "y" . }}
  {{end}}
  ```

  After:

  ```
  {{ if .Values.a }}
    {{ $x := 1 }}
  {{ else if .Values.b }}
    {{ include "x" . }}
  {{ else }}
    {{ include "y" . }}
  {{ end }}
  ```

- Inline control (start and end on the same line):

  ```
  {{ if .Values.enabled }} kind: ConfigMap {{ end }}
  ```

  Indentation for that line is at the current depth (no extra increase).

- Comment before the token:
  Before:
  ```
  {{/* leading comment */}}{{- if .Values.enabled }}
  {{ $x := 1 }}
  {{ end }}
  ```
  After:
  ```
  {{/* leading comment */}}{{- if .Values.enabled }}
    {{ $x := 1 }}
  {{ end }}
  ```

Note: Plain YAML lines without tags are not changed.

## How it works

- Processes files line-by-line while skipping an optional leading block comment
- Detects the token type at the start of the line (after any leading comment)
- Computes depth and indentation:
  - Renders the current token at the current depth (else/end render one level less)
  - After rendering: control start increases depth; end decreases depth
- Handles multi-line tags and inline constructs; comments within tags are safely skipped when searching for closing braces

## Limitations and notes

- Indentation step is fixed to 2 spaces
- Only lines with Go-template tags are touched; pure YAML indentation is not altered
- Supported “simple” functions are limited to include, fail, toYaml, printf
- Read/write errors for individual files are logged; processing continues. The final “Processed/Updated/Errors” summarizes results

Exit codes:

- 2 — invalid arguments
- 1 — critical error (e.g., templates directory doesn’t exist)
- 0 — otherwise (even if some files failed; they are counted in Errors)

## Roadmap / Ideas

- Configurable indent width (e.g., --indent)
- Check-only mode without writing (--check/--diff)
- Broader set of simple functions (tpl, required, dict, etc.) and keywords
- Golden tests and examples derived from cases.md
- Optional alignment for YAML lines around template tags

## Build from source

```
git clone https://github.com/digitalstudium/helmfmt
cd helmfmt
go build -o helmfmt
./helmfmt <chart-path>
```

Issues and PRs welcome!
