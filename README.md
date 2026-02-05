# helmfmt

`helmfmt` creates consistent formatting for Helm templates. It auto-aligns Go-template indentation for control blocks (e.g., `{{ if ... }}`, `{{ range ... }}`), variables, and comments, improving readability. It does **not** affect raw YAML structure, keeping your chart valid while making it cleaner.

It can be configured for [Zed IDE](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#zed-ide-configuration)/[VS Code](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#vscode-ide-configuration)/[VIM](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#vim-configuration), and as a [pre-commit hook](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#pre-commit-hook-configuration).

---

## What gets formatted

These Go-template tags are indented:

- Control blocks: `range`, `with`, `define`, `block`
- Branching: `if`, `else`, `else if`, `end`
- Vars: `{{ $var := ... }}`
- Some functions: `fail`, `printf` etc.
- Comments: `{{/* ... */}}`

These are not indented by default but can be [configured](https://github.com/digitalstudium/helmfmt?tab=readme-ov-file#configuration):

- `tpl`, `template`, `include` and `toYaml` because they can break YAML indentation

---

## Example

**Before**

```go-template
{{- range $cluster, $teams := .Values.clusters -}}
{{- range $team, $namespaces := $teams -}}
{{- range $namespace, $charts := $namespaces -}}
{{- range $chart, $config := $charts -}}
{{- range $release := $config.releases -}}
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{ include "generateAppName" (list $cluster $team $namespace $chart $release) }}
  namespace: argocd
spec:
  project: default
  source:
    repoURL: {{ $config.repo }}
    targetRevision: {{ $config.version }}
    chart: {{ $chart }}
  destination:
{{- with (get $.Values.clusterServers $cluster) }}
    server: {{ . }}
{{- else }}
{{- fail (printf "Cluster %s not found in clusterServers" $cluster) }}
{{- end }}
    namespace: {{ include "destinationNamespace" (list $team $namespace) }}
---
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
```

**After**

```go-template
{{- range $cluster, $teams := .Values.clusters -}}
  {{- range $team, $namespaces := $teams -}}
    {{- range $namespace, $charts := $namespaces -}}
      {{- range $chart, $config := $charts -}}
        {{- range $release := $config.releases -}}
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{ include "generateAppName" (list $cluster $team $namespace $chart $release) }}
  namespace: argocd
spec:
  project: default
  source:
    repoURL: {{ $config.repo }}
    targetRevision: {{ $config.version }}
    chart: {{ $chart }}
  destination:
    {{- with (get $.Values.clusterServers $cluster) }}
    server: {{ . }}
    {{- else }}
      {{- fail (printf "Cluster %s not found in clusterServers" $cluster) }}
    {{- end }}
    namespace: {{ include "destinationNamespace" (list $team $namespace) }}
---
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
```

## Installation

### One-click installation (Linux)
```bash
curl -L https://github.com/digitalstudium/helmfmt/releases/latest/download/helmfmt_Linux_x86_64.tar.gz | sudo tar -xzf - -C /usr/local/bin/ helmfmt
```
### Other methods
#### First method

Download it from [releases](https://github.com/digitalstudium/helmfmt/releases) and put into `PATH` folder, e. g. for Linux:
```bash
sudo install ./Downloads/helmfmt /usr/local/bin/
```

#### Second method

```bash
go install github.com/digitalstudium/helmfmt@latest
```

Then add `$HOME/go/bin/` to `PATH` if not already done:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc && source ~/.bashrc
```

#### Third method

```bash
git clone https://github.com/digitalstudium/helmfmt
cd helmfmt
go build
```

Then add it to `PATH` via:

```bash
sudo install ./helmfmt /usr/local/bin/
```


---

## Usage

```bash
helmfmt <chart-path>
helmfmt --files <file1> <file2> ...
helmfmt --files <file1> <file2> ... --stdout
helmfmt --enable-indent=toYaml,include --files <file1> <file2> ...
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

## Configuration

`helmfmt` can be configured using a `.helmfmt` file in JSON format. The tool looks for configuration files in this order:

1. `~/.helmfmt` (home directory)
2. `./.helmfmt` (project directory, overrides home directory configuration)

### Default Configuration

```json
{
  "indent_size": 2,
  "extensions": [".yaml", ".yml", ".tpl"],
  "rules": {
    "indent": {
      "tpl": {
        "disabled": true,
        "exclude": []
      },
      "toYaml": {
        "disabled": true,
        "exclude": []
      },
      "template": {
        "disabled": true,
        "exclude": []
      },
      "include": {
        "disabled": true,
        "exclude": []
      },
      "printf": {
        "disabled": false,
        "exclude": []
      },
      "fail": {
        "disabled": false,
        "exclude": []
      }
    }
  }
}
```

### Rule Configuration

Each rule can be configured with:

- **`disabled`**: Set to `true` to disable the rule entirely
- **`exclude`**: Array of file patterns to exclude from this rule

### Example Configurations

**Enable `tpl` and `toYaml` indentation:**

```json
{
  "rules": {
    "indent": {
      "tpl": {
        "disabled": false
      },
      "toYaml": {
        "disabled": false
      }
    }
  }
}
```

**Exclude test files from `include` indentation:**

```json
{
  "rules": {
    "indent": {
      "include": {
        "disabled": false,
        "exclude": ["tests/*", "**/test-*.yaml"]
      }
    }
  }
}
```

**Use 4 spaces for indentation:**

```json
{
  "indent_size": 4
}
```

### Command-line Rule Overrides

You can override configuration rules using command-line flags:

```bash
# Enable specific rules (overrides config file)
helmfmt --enable-indent=tpl,toYaml <chart-path>
```

---

## pre-commit hook configuration

https://github.com/pre-commit/pre-commit should be installed  first

To use `helmfmt` as a pre-commit hook, add the following to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/digitalstudium/helmfmt
    rev: v0.4.2
    hooks:
      - id: helmfmt
```

## Zed IDE configuration
First, you should install:

- helm language extension https://github.com/cabrinha/helm.zed
- and helm language server https://github.com/mrjosh/helm-ls

Then add these lines to your `settings.json`:

```json
  "file_types": {
    "Helm": [
      "**/templates/**/*.tpl",
      "**/templates/**/*.yaml",
      "**/templates/**/*.yml"
    ]
  },
  "languages": {
    "Helm": {
      "formatter": {
        "external": {
          "command": "helmfmt",
        }
      }
    }
  }
```

## VSCode IDE configuration

First, these extensions should be installed:

- https://github.com/WebFreak001/vscode-advanced-local-formatters
- https://github.com/vscode-kubernetes-tools/vscode-kubernetes-tools

Then add these lines to your settings:

```json
  "advancedLocalFormatters.formatters": [
    {
      "command": ["helmfmt"],
      "languages": ["helm"]
    }
  ],
  "[helm]": {
    "editor.defaultFormatter": "webfreak.advanced-local-formatters"
  }
```

## VIM configuration

### First method

Just type `:%! helmfmt` inside template

### Second method

Add to your `.vimrc`:

```vim
autocmd BufRead,BufNewFile */templates/*.yaml,*/templates/*.yml set filetype=helm
autocmd FileType helm nnoremap <buffer> <leader>f :%!helmfmt<CR>
```

Press `\f` in any Helm template to format it.

### Third method

Install [`towolf/vim-helm`](https://github.com/towolf/vim-helm) plugin for enhanced syntax highlighting. The filetype is set automatically. Just add to `.vimrc`:

```vim
autocmd FileType helm nnoremap <buffer> <leader>f :%!helmfmt<CR>
```

Press `\f` in any Helm template to format it.

---

## Roadmap

- More Helm funcs (dict, etc.)
- Format spaces inside tags

---

**Issues and PRs welcome!**
