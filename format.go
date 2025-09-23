package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	_ "unsafe"

	_ "helm.sh/helm/v3/pkg/engine" // Import to work with Helm's private functions (via go linkname)
)

// Типы токенов для ясности, что мы нашли
type tokenKind int

const (
	tokNone          tokenKind = iota
	tokControlOpen             // if, range, with, define, block
	tokElse                    // else/else if
	tokEnd                     // end
	tokVar                     // {{ $var ... }}
	tokSimple                  // include, fail, printf и другие простые функции
	tokControlInline           // управляющая конструкция с end в той же строке
)

var (
	// Основные паттерны для парсинга тегов
	tagOpenRe     = regexp.MustCompile(`^\s*\{\{(-?)(\s*)`)
	commentOpenRe = regexp.MustCompile(`^\s*(\{\{/\*|\{\{-\s/\*)`)
	commentEndRe  = regexp.MustCompile(`\*\/(?:}}|\s-}})`)

	// Паттерны для определения типов токенов
	varRe       = regexp.MustCompile(`^\s*\$`)
	controlRe   = regexp.MustCompile(`^\s*(if|range|with|define|block)\b`)
	elseRe      = regexp.MustCompile(`^\s*else\b`)
	endRe       = regexp.MustCompile(`^\s*end\b`)
	endInLineRe = regexp.MustCompile(`\{\{(-?)(\s*)end\b[^}]*(-?)\}\}`)

	// Для извлечения первого слова
	firstWordRe = regexp.MustCompile(`^\s*(\w+)`)
)

//go:linkname helmFuncMap helm.sh/helm/v3/pkg/engine.funcMap
func helmFuncMap() template.FuncMap

// validateTemplateSyntax validates the given template source string using
// Helm function set. Returns an error if the template has invalid syntax.
func validateTemplateSyntax(src string) error {

	// Get Helm's built-in function map
	helmFuncMap := helmFuncMap()

	// Create and parse template with helm function map
	_, err := template.New("validation").Funcs(helmFuncMap).Parse(src)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	return nil
}

// Главная функция выравнивания
func formatIndentation(src string, config *Config, filePath string) string {
	lines := strings.Split(src, "\n")
	depth := 0

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}

		// Сначала проверяем, есть ли комментарий в начале строки
		commentStart := i
		if commentOpenRe.MatchString(lines[i]) {
			// Это комментарий, найдем его конец
			_, _, _ = skipLeadingBlockComment(lines, i)
		}

		keyword, _, endLine, kind, found := getTokenAtLineStartSkippingLeadingComments(lines, i, config)
		if !found {
			continue
		}

		// Check if we should skip indenting this simple function
		if kind == tokSimple {
			ruleName := getRuleName(keyword, kind)
			if ruleName != "" {
				rule := config.Rules.Indent[ruleName] // Updated access pattern
				if rule.Disabled || matchesExcludePattern(filePath, rule.Exclude) {
					i = endLine
					continue // Skip indenting this token
				}
			}
		}

		// Вычисляем уровень отступа
		level := depth
		if kind == tokElse || kind == tokEnd {
			if level > 0 {
				level--
			}
		}

		// Apply indentation
		indent := strings.Repeat(" ", level*config.IndentSize)
		for j := commentStart; j <= endLine && j < len(lines); j++ {
			lines[j] = indent + strings.TrimLeft(lines[j], " \t")
		}

		// Always update depth for control structures
		switch kind {
		case tokControlOpen:
			depth++
		case tokEnd:
			if depth > 0 {
				depth--
			}
		}

		i = endLine
	}

	return strings.Join(lines, "\n")
}

// Ищем токен в начале строки, пропуская ведущий комментарий
func getTokenAtLineStartSkippingLeadingComments(lines []string, start int, config *Config) (keyword string, startLine int, endLine int, kind tokenKind, found bool) {
	i := start

	for {
		if i >= len(lines) {
			return "", start, start, tokNone, false
		}

		line := lines[i]

		// Проверяем, начинается ли строка с тега
		if !tagOpenRe.MatchString(line) {
			return "", start, start, tokNone, false
		}

		// Проверяем, начинается ли с комментария
		if commentOpenRe.MatchString(line) {
			ci, remainder, ok := skipLeadingBlockComment(lines, i)
			if !ok {
				return "", start, ci, tokNone, false
			}

			// Проверяем остаток строки после комментария
			remainder = strings.TrimLeft(remainder, " \t")
			if remainder == "" {
				i = ci + 1
				continue
			}

			if !tagOpenRe.MatchString(remainder) {
				return "", start, ci, tokNone, false
			}

			// Парсим токен из остатка строки
			return parseTokenFromLine(lines, ci, remainder, config)
		}

		// Парсим токен напрямую
		return parseTokenFromLine(lines, i, line, config)
	}
}

// Парсинг токена из строки
func parseTokenFromLine(lines []string, lineIdx int, line string, config *Config) (keyword string, startLine int, endLine int, kind tokenKind, found bool) {
	// Извлекаем содержимое после {{ или {{-
	match := tagOpenRe.FindStringSubmatch(line)
	if match == nil {
		return "", lineIdx, lineIdx, tokNone, false
	}

	content := line[len(match[0]):]

	// Проверяем тип токена
	switch {
	case varRe.MatchString(content):
		end := findTagEndMultiline(lines, lineIdx, line)
		return "$", lineIdx, end, tokVar, true

	case controlRe.MatchString(content):
		matches := controlRe.FindStringSubmatch(content)
		keyword := matches[1]

		end := findTagEndMultiline(lines, lineIdx, line)

		// Проверяем наличие end в той же строке/блоке
		if hasEndInRange(lines, lineIdx, end) {
			return keyword, lineIdx, end, tokControlInline, true
		}

		return keyword, lineIdx, end, tokControlOpen, true

	case elseRe.MatchString(content):
		end := findTagEndMultiline(lines, lineIdx, line)
		return "else", lineIdx, end, tokElse, true

	case endRe.MatchString(content):
		end := findTagEndMultiline(lines, lineIdx, line)
		return "end", lineIdx, end, tokEnd, true

	default:
		// Check if first word has a rule defined - if so, treat as simple
		if matches := firstWordRe.FindStringSubmatch(content); matches != nil {
			keyword := matches[1]

			if _, hasRule := config.Rules.Indent[keyword]; hasRule { // Updated access pattern
				end := findTagEndMultiline(lines, lineIdx, line)
				return keyword, lineIdx, end, tokSimple, true
			}
		}
	}

	return "", lineIdx, lineIdx, tokNone, false
}

// Пропуск блочного комментария
func skipLeadingBlockComment(lines []string, start int) (endLine int, remainder string, ok bool) {
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if i == start {
			// Ищем закрытие комментария в первой строке
			if match := commentEndRe.FindStringIndex(line); match != nil {
				return i, line[match[1]:], true
			}
		} else {
			// В последующих строках ищем закрытие с начала
			line = strings.TrimLeft(line, " \t")
			if match := commentEndRe.FindStringIndex(line); match != nil {
				return i, line[match[1]:], true
			}
		}
	}
	return start, "", false
}

// Поиск закрытия тега с возможностью многострочности
func findTagEndMultiline(lines []string, start int, firstLine string) int {
	inComment := false

	for i := start; i < len(lines); i++ {
		line := lines[i]
		if i > start {
			line = strings.TrimLeft(line, " \t")
		}

		pos := 0
		if i == start {
			// Пропускаем начало тега в первой строке
			if match := tagOpenRe.FindStringIndex(firstLine); match != nil {
				pos = match[1]
			}
		}

		for pos < len(line) {
			if inComment {
				if idx := strings.Index(line[pos:], "*/"); idx >= 0 {
					pos += idx + 2
					inComment = false
					continue
				}
				break // комментарий продолжается на следующей строке
			}

			// Проверяем начало комментария
			if pos+1 < len(line) && line[pos] == '/' && line[pos+1] == '*' {
				inComment = true
				pos += 2
				continue
			}

			// Проверяем закрытие тега
			if pos+1 < len(line) && line[pos] == '}' && line[pos+1] == '}' {
				return i
			}
			if pos+2 < len(line) && line[pos] == '-' && line[pos+1] == '}' && line[pos+2] == '}' {
				return i
			}

			pos++
		}
	}

	return len(lines) - 1
}

// Проверка наличия end в диапазоне строк
func hasEndInRange(lines []string, start, end int) bool {
	for i := start; i <= end && i < len(lines); i++ {
		if endInLineRe.MatchString(lines[i]) {
			return true
		}
	}
	return false
}

func matchesExcludePattern(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		// Convert glob pattern to regex
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}

		// Also support regex patterns
		if regexp.MustCompile(pattern).MatchString(filePath) {
			return true
		}
	}
	return false
}

func getRuleName(keyword string, kind tokenKind) string {
	if kind == tokSimple {
		return keyword // Just return the keyword (printf, include, fail)
	}
	return ""
}
