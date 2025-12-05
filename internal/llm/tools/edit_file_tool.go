// go
// file: internal/llm/tools/edit_file_tool.go
package tools

import (
	"context"
	"errors"
	"fmt"
	"narrabyte/internal/events"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type EditInput struct {
	// Repository must be "docs" - editing the code repository is not allowed.
	Repository Repository `json:"repository" jsonschema:"enum=docs,description=Must be 'docs' - editing the code repository is not allowed"`
	// FilePath is the relative path to the file within the docs repository.
	FilePath string `json:"file_path" jsonschema:"description=The path to the file relative to the docs repository root (e.g. 'api/endpoints.md'). NEVER use absolute paths."`
	// OldString is the exact text to replace. Leave empty to overwrite the entire file.
	OldString string `json:"old_string" jsonschema:"description=The exact text to replace. Leave empty to overwrite the entire file with new_string."`
	// NewString is the replacement text.
	NewString string `json:"new_string" jsonschema:"description=The replacement text"`
	// ReplaceAll replaces all occurrences of old_string instead of just the first.
	ReplaceAll bool `json:"replace_all,omitempty" jsonschema:"description=Replace all occurrences of old_string instead of a single instance"`
}

type EditOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

const (
	singleCandidateSimilarityThreshold    = 0.0
	multipleCandidatesSimilarityThreshold = 0.3
)

var (
	errOldStringNotFound = errors.New("old_string not found in content")
	errMultipleMatches   = errors.New("multiple matches for old_string")
)

type replaceError struct {
	err         error
	occurrences int
}

func (e *replaceError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *replaceError) Unwrap() error {
	return e.err
}

func Edit(ctx context.Context, in *EditInput) (*EditOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Edit: starting"))

	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: input is required"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	// Enforce docs-only repository
	if in.Repository != RepositoryDocs {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Edit: repository must be 'docs', got '%s'", in.Repository)))
		return &EditOutput{
			Title:  "",
			Output: fmt.Sprintf("Format error: editing is only allowed in the 'docs' repository, got '%s'", in.Repository),
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	pathArg := strings.TrimSpace(in.FilePath)
	if pathArg == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: file_path is required"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	if strings.TrimSpace(in.OldString) != "" && in.OldString == in.NewString {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: old_string and new_string must be different"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: old_string and new_string must be different",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	// Resolve path using the repository-scoped resolver
	abs, err := ResolveRepositoryPath(ctx, in.Repository, pathArg)
	if err != nil {
		displayPath := FormatDisplayPath(in.Repository, pathArg)
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Edit: %v", err)))
		return &EditOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: %v", err),
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	displayPath := FormatDisplayPath(in.Repository, pathArg)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: target '%s'", displayPath)))

	dir := filepath.Dir(abs)
	if st, derr := os.Stat(dir); derr != nil || !st.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: directory does not exist"))
		return &EditOutput{
			Title:  displayPath,
			Output: "Format error: directory does not exist",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	if st, err := os.Stat(abs); err == nil && st.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: cannot edit directory"))
		return &EditOutput{
			Title:  displayPath,
			Output: "Format error: cannot edit directory",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	if st, err := os.Stat(abs); err == nil && !st.IsDir() {
		if bin, berr := isBinaryFile(abs); berr == nil && bin {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: cannot edit binary file"))
			return &EditOutput{
				Title:  displayPath,
				Output: "Format error: cannot edit binary file",
				Metadata: map[string]string{
					"error":       "format_error",
					"replaced":    "false",
					"occurrences": "0",
				},
			}, nil
		}
	}

	var contentOld string
	var contentNew string
	replacedCount := 1

	if in.OldString == "" {
		if existing, readErr := os.ReadFile(abs); readErr == nil {
			contentOld = string(existing)
		}
		contentNew = in.NewString
	} else {
		contentBytes, rerr := os.ReadFile(abs)
		if rerr != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: file does not exist"))
			return &EditOutput{
				Title:  displayPath,
				Output: "Format error: file does not exist",
				Metadata: map[string]string{
					"error":       "format_error",
					"replaced":    "false",
					"occurrences": "0",
				},
			}, nil
		}
		contentOld = string(contentBytes)
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: read %d bytes", len(contentBytes))))

		replaced, occ, repErr := replaceContent(contentOld, in.OldString, in.NewString, in.ReplaceAll)
		if repErr != nil {
			switch {
			case errors.Is(repErr, errOldStringNotFound):
				events.Emit(ctx, events.LLMEventTool, events.NewInfo("Edit: old_string not found"))
				return &EditOutput{
					Title:  displayPath,
					Output: "Edit error: old_string not found in content",
					Metadata: map[string]string{
						"error":       "search_not_found",
						"replaced":    "false",
						"occurrences": "0",
					},
				}, nil
			case errors.Is(repErr, errMultipleMatches):
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("Edit: ambiguous match without replace_all"))
				return &EditOutput{
					Title:  displayPath,
					Output: "Edit error: old_string found multiple times and requires more code context to uniquely identify the intended match",
					Metadata: map[string]string{
						"error":       "ambiguous_match",
						"replaced":    "false",
						"occurrences": fmt.Sprintf("%d", occ),
					},
				}, nil
			default:
				return nil, repErr
			}
		}
		contentNew = replaced
		replacedCount = occ
	}

	if err := os.WriteFile(abs, []byte(contentNew), 0o644); err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Edit: write error: %v", err)))
		return nil, err
	}

	diff := computeDiff(displayPath, contentOld, contentNew)
	additions, deletions := diffLineCounts(contentOld, contentNew)

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: replaced %d occurrence(s)", replacedCount)))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("Edit: done for '%s'", displayPath), "edit", displayPath))

	return &EditOutput{
		Title:  displayPath,
		Output: "Edit success",
		Metadata: map[string]string{
			"error":       "",
			"replaced":    "true",
			"occurrences": fmt.Sprintf("%d", replacedCount),
			"diff":        diff,
			"additions":   fmt.Sprintf("%d", additions),
			"deletions":   fmt.Sprintf("%d", deletions),
		},
	}, nil
}

func replaceContent(content, old, newVal string, replaceAll bool) (string, int, error) {
	if old == newVal {
		return "", 0, fmt.Errorf("old_string and new_string must be different")
	}
	if old == "" {
		return newVal, 1, nil
	}

	notFound := true
	ambiguousOccurrences := 0

	for _, replacer := range replacers() {
		for _, search := range replacer(content, old) {
			index := strings.Index(content, search)
			if index == -1 {
				continue
			}
			notFound = false

			occurrenceCount := strings.Count(content, search)
			if replaceAll {
				return strings.ReplaceAll(content, search, newVal), occurrenceCount, nil
			}

			if occurrenceCount > 1 {
				ambiguousOccurrences = occurrenceCount
				continue
			}

			return content[:index] + newVal + content[index+len(search):], 1, nil
		}
	}

	if notFound {
		return "", 0, &replaceError{err: errOldStringNotFound}
	}
	if ambiguousOccurrences == 0 {
		ambiguousOccurrences = 2
	}
	return "", ambiguousOccurrences, &replaceError{err: errMultipleMatches, occurrences: ambiguousOccurrences}
}

type replacer func(content, find string) []string

func replacers() []replacer {
	return []replacer{
		simpleReplacer,
		lineTrimmedReplacer,
		blockAnchorReplacer,
		whitespaceNormalizedReplacer,
		indentationFlexibleReplacer,
		escapeNormalizedReplacer,
		trimmedBoundaryReplacer,
		contextAwareReplacer,
		multiOccurrenceReplacer,
	}
}

func simpleReplacer(_ string, find string) []string {
	if find == "" {
		return nil
	}
	return []string{find}
}

func lineTrimmedReplacer(content, find string) []string {
	originalLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")
	if len(searchLines) == 0 {
		return nil
	}
	if searchLines[len(searchLines)-1] == "" {
		searchLines = searchLines[:len(searchLines)-1]
	}
	if len(searchLines) == 0 {
		return nil
	}

	var matches []string
	for i := 0; i <= len(originalLines)-len(searchLines); i++ {
		match := true
		for j := 0; j < len(searchLines); j++ {
			if strings.TrimSpace(originalLines[i+j]) != strings.TrimSpace(searchLines[j]) {
				match = false
				break
			}
		}
		if match {
			start, end := lineRangeIndices(originalLines, i, i+len(searchLines)-1)
			matches = append(matches, content[start:end])
		}
	}
	return matches
}

func blockAnchorReplacer(content, find string) []string {
	originalLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")

	if len(searchLines) < 3 {
		return nil
	}
	if searchLines[len(searchLines)-1] == "" {
		searchLines = searchLines[:len(searchLines)-1]
	}
	if len(searchLines) < 3 {
		return nil
	}

	firstLineSearch := strings.TrimSpace(searchLines[0])
	lastLineSearch := strings.TrimSpace(searchLines[len(searchLines)-1])
	searchBlockSize := len(searchLines)

	type candidate struct {
		startLine int
		endLine   int
	}

	var candidates []candidate
	for i := 0; i < len(originalLines); i++ {
		if strings.TrimSpace(originalLines[i]) != firstLineSearch {
			continue
		}
		for j := i + 2; j < len(originalLines); j++ {
			if strings.TrimSpace(originalLines[j]) == lastLineSearch {
				candidates = append(candidates, candidate{startLine: i, endLine: j})
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	if len(candidates) == 1 {
		c := candidates[0]
		actualBlockSize := c.endLine - c.startLine + 1
		similarity := 0.0
		linesToCheck := min(searchBlockSize-2, actualBlockSize-2)

		if linesToCheck > 0 {
			for j := 1; j < searchBlockSize-1 && j < actualBlockSize-1; j++ {
				originalLine := strings.TrimSpace(originalLines[c.startLine+j])
				searchLine := strings.TrimSpace(searchLines[j])
				maxLen := max(len(originalLine), len(searchLine))
				if maxLen == 0 {
					continue
				}
				distance := levenshtein(originalLine, searchLine)
				similarity += (1 - float64(distance)/float64(maxLen)) / float64(linesToCheck)
				if similarity >= singleCandidateSimilarityThreshold {
					break
				}
			}
		} else {
			similarity = 1.0
		}

		if similarity >= singleCandidateSimilarityThreshold {
			start, end := lineRangeIndices(originalLines, c.startLine, c.endLine)
			return []string{content[start:end]}
		}
		return nil
	}

	var (
		bestMatch     *candidate
		maxSimilarity = -1.0
	)

	for idx := range candidates {
		c := candidates[idx]
		actualBlockSize := c.endLine - c.startLine + 1
		linesToCheck := min(searchBlockSize-2, actualBlockSize-2)
		similarity := 0.0

		if linesToCheck > 0 {
			for j := 1; j < searchBlockSize-1 && j < actualBlockSize-1; j++ {
				originalLine := strings.TrimSpace(originalLines[c.startLine+j])
				searchLine := strings.TrimSpace(searchLines[j])
				maxLen := max(len(originalLine), len(searchLine))
				if maxLen == 0 {
					continue
				}
				distance := levenshtein(originalLine, searchLine)
				similarity += 1 - float64(distance)/float64(maxLen)
			}
			similarity /= float64(linesToCheck)
		} else {
			similarity = 1.0
		}

		if similarity > maxSimilarity {
			maxSimilarity = similarity
			bestMatch = &c
		}
	}

	if bestMatch != nil && maxSimilarity >= multipleCandidatesSimilarityThreshold {
		start, end := lineRangeIndices(originalLines, bestMatch.startLine, bestMatch.endLine)
		return []string{content[start:end]}
	}

	return nil
}

func whitespaceNormalizedReplacer(content, find string) []string {
	normalize := func(text string) string {
		parts := strings.Fields(text)
		return strings.TrimSpace(strings.Join(parts, " "))
	}
	normalizedFind := normalize(find)
	if normalizedFind == "" {
		return nil
	}

	var matches []string
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		normalizedLine := normalize(line)
		if normalizedLine == normalizedFind {
			matches = append(matches, line)
			continue
		}
		if strings.Contains(normalizedLine, normalizedFind) {
			words := strings.Fields(strings.TrimSpace(find))
			if len(words) == 0 {
				continue
			}
			for idx := range words {
				words[idx] = regexp.QuoteMeta(words[idx])
			}
			pattern := strings.Join(words, `\s+`)
			regex, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			if loc := regex.FindString(line); loc != "" {
				matches = append(matches, loc)
			}
		}
	}

	findLines := strings.Split(find, "\n")
	if len(findLines) > 1 {
		for i := 0; i <= len(lines)-len(findLines); i++ {
			block := strings.Join(lines[i:i+len(findLines)], "\n")
			if normalize(block) == normalizedFind {
				matches = append(matches, block)
			}
		}
	}

	return matches
}

func indentationFlexibleReplacer(content, find string) []string {
	removeIndentation := func(text string) string {
		lines := strings.Split(text, "\n")
		var nonEmpty []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmpty = append(nonEmpty, line)
			}
		}
		if len(nonEmpty) == 0 {
			return text
		}

		minIndent := -1
		for _, line := range nonEmpty {
			indent := leadingWhitespace(line)
			if minIndent == -1 || indent < minIndent {
				minIndent = indent
			}
		}
		if minIndent <= 0 {
			return text
		}

		for i := range lines {
			if strings.TrimSpace(lines[i]) == "" {
				continue
			}
			if len(lines[i]) >= minIndent {
				lines[i] = lines[i][minIndent:]
			}
		}
		return strings.Join(lines, "\n")
	}

	normalizedFind := removeIndentation(find)
	contentLines := strings.Split(content, "\n")
	findLines := strings.Split(find, "\n")
	if len(findLines) == 0 {
		return nil
	}

	var matches []string
	for i := 0; i <= len(contentLines)-len(findLines); i++ {
		block := strings.Join(contentLines[i:i+len(findLines)], "\n")
		if removeIndentation(block) == normalizedFind {
			matches = append(matches, block)
		}
	}
	return matches
}

func escapeNormalizedReplacer(content, find string) []string {
	unescape := func(str string) string {
		var b strings.Builder
		for i := 0; i < len(str); i++ {
			if str[i] == '\\' && i+1 < len(str) {
				switch str[i+1] {
				case 'n':
					b.WriteByte('\n')
					i++
					continue
				case 't':
					b.WriteByte('\t')
					i++
					continue
				case 'r':
					b.WriteByte('\r')
					i++
					continue
				case '\'':
					b.WriteByte('\'')
					i++
					continue
				case '"':
					b.WriteByte('"')
					i++
					continue
				case '`':
					b.WriteByte('`')
					i++
					continue
				case '\\':
					b.WriteByte('\\')
					i++
					continue
				case '\n':
					b.WriteByte('\n')
					i++
					continue
				case '$':
					b.WriteByte('$')
					i++
					continue
				}
			}
			b.WriteByte(str[i])
		}
		return b.String()
	}

	unescapedFind := unescape(find)
	var matches []string
	if strings.Contains(content, unescapedFind) {
		matches = append(matches, unescapedFind)
	}

	lines := strings.Split(content, "\n")
	findLines := strings.Split(unescapedFind, "\n")
	for i := 0; i <= len(lines)-len(findLines); i++ {
		block := strings.Join(lines[i:i+len(findLines)], "\n")
		if unescape(block) == unescapedFind {
			matches = append(matches, block)
		}
	}
	return matches
}

func multiOccurrenceReplacer(content, find string) []string {
	if find == "" {
		return nil
	}
	var matches []string
	startIndex := 0
	for {
		idx := strings.Index(content[startIndex:], find)
		if idx == -1 {
			break
		}
		matches = append(matches, find)
		startIndex += idx + len(find)
		if len(find) == 0 {
			break
		}
	}
	return matches
}

func trimmedBoundaryReplacer(content, find string) []string {
	trimmedFind := strings.TrimSpace(find)
	if trimmedFind == "" || trimmedFind == find {
		return nil
	}

	var matches []string
	if strings.Contains(content, trimmedFind) {
		matches = append(matches, trimmedFind)
	}

	lines := strings.Split(content, "\n")
	findLines := strings.Split(find, "\n")
	for i := 0; i <= len(lines)-len(findLines); i++ {
		block := strings.Join(lines[i:i+len(findLines)], "\n")
		if strings.TrimSpace(block) == trimmedFind {
			matches = append(matches, block)
		}
	}
	return matches
}

func contextAwareReplacer(content, find string) []string {
	findLines := strings.Split(find, "\n")
	if len(findLines) < 3 {
		return nil
	}
	if findLines[len(findLines)-1] == "" {
		findLines = findLines[:len(findLines)-1]
	}
	if len(findLines) < 3 {
		return nil
	}

	firstLine := strings.TrimSpace(findLines[0])
	lastLine := strings.TrimSpace(findLines[len(findLines)-1])
	contentLines := strings.Split(content, "\n")

	for i := 0; i < len(contentLines); i++ {
		if strings.TrimSpace(contentLines[i]) != firstLine {
			continue
		}
		for j := i + 2; j < len(contentLines); j++ {
			if strings.TrimSpace(contentLines[j]) != lastLine {
				continue
			}
			blockLines := contentLines[i : j+1]
			if len(blockLines) != len(findLines) {
				break
			}

			matchingLines := 0
			totalNonEmpty := 0
			for k := 1; k < len(blockLines)-1; k++ {
				blockLine := strings.TrimSpace(blockLines[k])
				findLine := strings.TrimSpace(findLines[k])
				if blockLine != "" || findLine != "" {
					totalNonEmpty++
					if blockLine == findLine {
						matchingLines++
					}
				}
			}

			if totalNonEmpty == 0 || float64(matchingLines)/float64(totalNonEmpty) >= 0.5 {
				start, end := lineRangeIndices(contentLines, i, j)
				return []string{content[start:end]}
			}
			break
		}
	}

	return nil
}

func computeDiff(path, before, after string) string {
	dmp := diffmatchpatch.New()
	patches := dmp.PatchMake(normalizeLineEndings(before), normalizeLineEndings(after))
	patchText := dmp.PatchToText(patches)
	header := fmt.Sprintf("--- %s\n+++ %s\n", path, path)
	return trimDiff(header + patchText)
}

func diffLineCounts(before, after string) (int, int) {
	dmp := diffmatchpatch.New()
	a, b, arr := dmp.DiffLinesToRunes(before, after)
	diffs := dmp.DiffMainRunes(a, b, false)
	dmp.DiffCharsToLines(diffs, arr)

	additions := 0
	deletions := 0
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			additions += diffLinesCount(d.Text)
		case diffmatchpatch.DiffDelete:
			deletions += diffLinesCount(d.Text)
		}
	}
	return additions, deletions
}

func diffLinesCount(text string) int {
	if text == "" {
		return 0
	}
	lines := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		lines++
	}
	return lines
}

func normalizeLineEndings(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func trimDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var contentLines []string
	for _, line := range lines {
		if (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) &&
			!strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "+++") {
			contentLines = append(contentLines, line)
		}
	}

	if len(contentLines) == 0 {
		return diff
	}

	minIndent := -1
	for _, line := range contentLines {
		content := line[1:]
		if strings.TrimSpace(content) == "" {
			continue
		}
		indent := leadingWhitespace(content)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return diff
	}

	for i, line := range lines {
		if (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) &&
			!strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "+++") {
			prefix := line[:1]
			content := line[1:]
			if len(content) >= minIndent {
				lines[i] = prefix + content[minIndent:]
			}
		}
	}

	return strings.Join(lines, "\n")
}

func lineRangeIndices(lines []string, startLine, endLine int) (int, int) {
	startIdx := 0
	for i := 0; i < startLine; i++ {
		startIdx += len(lines[i])
		if i < len(lines)-1 {
			startIdx++
		}
	}

	endIdx := startIdx
	for i := startLine; i <= endLine && i < len(lines); i++ {
		endIdx += len(lines[i])
		if i < endLine && i < len(lines)-1 {
			endIdx++
		}
	}

	return startIdx, endIdx
}

func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}

	matrix := make([][]int, len(ar)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(br)+1)
	}
	for i := 0; i <= len(ar); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(br); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(ar); i++ {
		for j := 1; j <= len(br); j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(ar)][len(br)]
}

func leadingWhitespace(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			count++
			continue
		}
		break
	}
	return count
}

func min(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, v := range values[1:] {
		if v < result {
			result = v
		}
	}
	return result
}

func max(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, v := range values[1:] {
		if v > result {
			result = v
		}
	}
	return result
}
