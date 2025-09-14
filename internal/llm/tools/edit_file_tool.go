package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EditInput defines parameters for the edit tool.
type EditInput struct {
	// FilePath is the path to the file to modify (absolute or relative to the project root).
	FilePath string `json:"file_path" jsonschema:"description=The path to the file to modify (absolute or relative to project root)"`
	// OldString is the text to replace. If empty, the entire file content will be replaced with NewString.
	OldString string `json:"old_string" jsonschema:"description=The text to replace (empty to overwrite entire file)"`
	// NewString is the replacement text.
	NewString string `json:"new_string" jsonschema:"description=The replacement text (must be different from old_string)"`
	// ReplaceAll replaces all occurrences of the matched search block.
	ReplaceAll bool `json:"replace_all,omitempty" jsonschema:"description=Replace all occurrences of the matched search block (default false)"`
}

// EditOutput is the output of the edit tool.
type EditOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Edit applies a context-aware replacement to a single file under the configured project root.
func Edit(_ context.Context, in *EditInput) (*EditOutput, error) {
	println("EditFile input: ", in.FilePath, in.OldString, in.NewString, in.ReplaceAll)
	if in == nil {
		return &EditOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	if in.OldString == in.NewString {
		return &EditOutput{
			Title:  "",
			Output: "Format error: old_string and new_string must be different",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	base, err := getListDirectoryBaseRoot()
	if err != nil {
		return &EditOutput{
			Title:  "",
			Output: "Format error: project root not set",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	p := strings.TrimSpace(in.FilePath)
	if p == "" {
		return &EditOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Resolve target path under base, ensuring it cannot escape.
	var absPath string
	if filepath.IsAbs(p) {
		// Ensure absolute path is under base
		absBase, err := filepath.Abs(base)
		if err != nil {
			return nil, err
		}
		absCandidate, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		relToBase, err := filepath.Rel(absBase, absCandidate)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			return &EditOutput{
				Title:  filepath.ToSlash(p),
				Output: "Format error: file is not in the configured project root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		absPath = absCandidate
	} else {
		abs, ok := safeJoinUnderBase(base, p)
		if !ok {
			return &EditOutput{
				Title:  filepath.ToSlash(p),
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		absPath = abs
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			rel, _ := filepath.Rel(base, absPath)
			rel = filepath.ToSlash(rel)
			return &EditOutput{
				Title:  rel,
				Output: fmt.Sprintf("Format error: directory does not exist: %s", dir),
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		rel, _ := filepath.Rel(base, absPath)
		rel = filepath.ToSlash(rel)
		return &EditOutput{
			Title:  rel,
			Output: fmt.Sprintf("Format error: not a directory: %s", dir),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Read old content (if file exists). If not, old content is empty.
	var existed bool
	var oldContent string
	if st, err := os.Stat(absPath); err == nil && !st.IsDir() {
		existed = true
		// Binary check for safety
		if bin, berr := isBinaryFile(absPath); berr == nil && bin {
			rel, _ := filepath.Rel(base, absPath)
			rel = filepath.ToSlash(rel)
			return &EditOutput{
				Title:  rel,
				Output: fmt.Sprintf("Format error: cannot edit binary file: %s", absPath),
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		data, rerr := os.ReadFile(absPath)
		if rerr != nil {
			return nil, rerr
		}
		oldContent = string(data)
	}

	var newContent string
	var replacedCount int
	var replaced bool

	if in.OldString == "" {
		// Overwrite entire content (or create the file)
		newContent = in.NewString
		// If content differs, consider as replaced
		replaced = (oldContent != newContent)
		if replaced {
			// Count as a single replacement of full content
			replacedCount = 1
		}
	} else {
		// Context-aware replacement path
		var out string
		out, replacedCount, err = replace(oldContent, in.OldString, in.NewString, in.ReplaceAll)
		if err != nil {
			// Convert search errors into structured tool output
			rel, _ := filepath.Rel(base, absPath)
			rel = filepath.ToSlash(rel)
			msg := err.Error()
			code := "input_error"
			switch msg {
			case "old_string not found in content", "oldString not found in content":
				code = "search_not_found"
			case "old_string found multiple times and requires more code context to uniquely identify the intended match",
				"oldString found multiple times and requires more code context to uniquely identify the intended match":
				code = "ambiguous_match"
			case "old_string and new_string must be different":
				code = "format_error"
			}
			return &EditOutput{
				Title:  rel,
				Output: "Edit error: " + msg,
				Metadata: map[string]string{
					"error": code,
				},
			}, nil
		}
		newContent = out
		replaced = replacedCount > 0
	}

	// Write new content
	if err := os.WriteFile(absPath, []byte(newContent), 0o644); err != nil {
		return nil, err
	}

	// Build a simple unified diff and trim it for readability
	diff := trimDiff(createTwoFilesPatch(absPath, absPath, oldContent, newContent))

	// Title relative to base
	rel, rerr := filepath.Rel(base, absPath)
	if rerr != nil {
		rel = absPath
	}
	rel = filepath.ToSlash(rel)

	// Output message
	verb := "Edited"
	if !existed {
		verb = "Created"
	}
	outMsg := fmt.Sprintf("%s file: %s", verb, rel)
	if replacedCount > 1 {
		outMsg += fmt.Sprintf(" (replaced %d occurrences)", replacedCount)
	} else if replacedCount == 1 {
		outMsg += " (replaced 1 occurrence)"
	} else if in.OldString != "" {
		outMsg += " (no changes)"
	}

	meta := map[string]string{
		"filepath":    rel,
		"replaced":    fmt.Sprintf("%v", replaced),
		"occurrences": fmt.Sprintf("%d", replacedCount),
		"diff":        diff,
	}

	return &EditOutput{
		Title:    rel,
		Output:   outMsg,
		Metadata: meta,
	}, nil
}

// --- Replacement logic (ported and adapted) ---

type replacer func(content, find string) []string

const (
	singleCandidateSimilarityThreshold    = 0.0
	multipleCandidatesSimilarityThreshold = 0.3
)

func levenshtein(a, b string) int {
	if a == "" || b == "" {
		if len(a) > len(b) {
			return len(a)
		}
		return len(b)
	}
	la, lb := len(a), len(b)
	dp := make([]int, (la+1)*(lb+1))
	at := func(i, j int) int { return i*(lb+1) + j }
	for i := 0; i <= la; i++ {
		dp[at(i, 0)] = i
	}
	for j := 0; j <= lb; j++ {
		dp[at(0, j)] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := dp[at(i-1, j)] + 1
			ins := dp[at(i, j-1)] + 1
			sub := dp[at(i-1, j-1)] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			dp[at(i, j)] = m
		}
	}
	return dp[at(la, lb)]
}

func simpleReplacer(_ string, find string) []string {
	return []string{find}
}

func lineTrimmedReplacer(content, find string) []string {
	origLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")
	if len(searchLines) > 0 && searchLines[len(searchLines)-1] == "" {
		searchLines = searchLines[:len(searchLines)-1]
	}
	if len(searchLines) == 0 {
		return nil
	}

	// Precompute line start offsets without overshooting the content length.
	offsets := make([]int, len(origLines)+1)
	pos := 0
	for i := 0; i < len(origLines); i++ {
		offsets[i] = pos
		pos += len(origLines[i])
		if i < len(origLines)-1 {
			pos += 1 // account for the newline between lines
		}
	}
	offsets[len(origLines)] = pos

	var out []string
	for i := 0; i <= len(origLines)-len(searchLines); i++ {
		matches := true
		for j := 0; j < len(searchLines); j++ {
			if strings.TrimSpace(origLines[i+j]) != strings.TrimSpace(searchLines[j]) {
				matches = false
				break
			}
		}
		if matches {
			start := offsets[i]
			end := offsets[i+len(searchLines)]
			if end > len(content) {
				end = len(content)
			}
			// Remove trailing newline that does not belong to the block
			if end > start && content[end-1] == '\n' {
				end--
			}
			if start >= 0 && start <= end && end <= len(content) {
				out = append(out, content[start:end])
			}
		}
	}
	return out
}

func blockAnchorReplacer(content, find string) []string {
	origLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")
	if len(searchLines) < 3 {
		return nil
	}
	if searchLines[len(searchLines)-1] == "" {
		searchLines = searchLines[:len(searchLines)-1]
	}

	first := strings.TrimSpace(searchLines[0])
	last := strings.TrimSpace(searchLines[len(searchLines)-1])
	searchBlockSize := len(searchLines)

	type rng struct{ startLine, endLine int }
	var candidates []rng
	for i := 0; i < len(origLines); i++ {
		if strings.TrimSpace(origLines[i]) != first {
			continue
		}
		for j := i + 2; j < len(origLines); j++ {
			if strings.TrimSpace(origLines[j]) == last {
				candidates = append(candidates, rng{startLine: i, endLine: j})
				break
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	// helper to slice substring by lines with safe offsets
	offsets := make([]int, len(origLines)+1)
	pos := 0
	for i := 0; i < len(origLines); i++ {
		offsets[i] = pos
		pos += len(origLines[i])
		if i < len(origLines)-1 {
			pos += 1
		}
	}
	offsets[len(origLines)] = pos

	toSubstr := func(s, e int) string {
		start := offsets[s]
		end := offsets[e+1]
		if end > len(content) {
			end = len(content)
		}
		if end > start && content[end-1] == '\n' {
			end--
		}
		if start < 0 {
			start = 0
		}
		if start > end {
			start = end
		}
		return content[start:end]
	}

	if len(candidates) == 1 {
		c := candidates[0]
		actualBlock := c.endLine - c.startLine + 1
		similarity := 0.0
		linesToCheck := min(searchBlockSize-2, actualBlock-2)
		if linesToCheck > 0 {
			for j := 1; j < searchBlockSize-1 && j < actualBlock-1; j++ {
				origLine := strings.TrimSpace(origLines[c.startLine+j])
				searchLine := strings.TrimSpace(searchLines[j])
				maxLen := max(len(origLine), len(searchLine))
				if maxLen == 0 {
					continue
				}
				d := levenshtein(origLine, searchLine)
				similarity += (1 - float64(d)/float64(maxLen)) / float64(linesToCheck)
				if similarity >= singleCandidateSimilarityThreshold {
					break
				}
			}
		} else {
			similarity = 1.0
		}
		if similarity >= singleCandidateSimilarityThreshold {
			return []string{toSubstr(c.startLine, c.endLine)}
		}
		return nil
	}

	// Multiple candidates: choose best by average similarity
	best := -1.0
	var bestR rng
	for _, c := range candidates {
		actualBlock := c.endLine - c.startLine + 1
		similarity := 0.0
		linesToCheck := min(searchBlockSize-2, actualBlock-2)
		if linesToCheck > 0 {
			for j := 1; j < searchBlockSize-1 && j < actualBlock-1; j++ {
				origLine := strings.TrimSpace(origLines[c.startLine+j])
				searchLine := strings.TrimSpace(searchLines[j])
				maxLen := max(len(origLine), len(searchLine))
				if maxLen == 0 {
					continue
				}
				d := levenshtein(origLine, searchLine)
				similarity += 1 - float64(d)/float64(maxLen)
			}
			similarity /= float64(linesToCheck)
		} else {
			similarity = 1.0
		}
		if similarity > best {
			best = similarity
			bestR = c
		}
	}
	if best >= multipleCandidatesSimilarityThreshold {
		return []string{toSubstr(bestR.startLine, bestR.endLine)}
	}
	return nil
}

func whitespaceNormalizedReplacer(content, find string) []string {
	normalize := func(s string) string {
		re := regexp.MustCompile(`\s+`)
		return strings.TrimSpace(re.ReplaceAllString(s, " "))
	}
	normalizedFind := normalize(find)
	var out []string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if normalize(line) == normalizedFind {
			out = append(out, line)
		} else {
			normLine := normalize(line)
			if strings.Contains(normLine, normalizedFind) {
				words := strings.Fields(strings.TrimSpace(find))
				if len(words) > 0 {
					var b strings.Builder
					for i, w := range words {
						if i > 0 {
							b.WriteString(`\\s+`)
						}
						// escape regex meta
						b.WriteString(regexp.QuoteMeta(w))
					}
					rx, err := regexp.Compile(b.String())
					if err == nil {
						if m := rx.FindString(line); m != "" {
							out = append(out, m)
						}
					}
				}
			}
		}
	}

	findLines := strings.Split(find, "\n")
	if len(findLines) > 1 {
		for i := 0; i <= len(lines)-len(findLines); i++ {
			block := strings.Join(lines[i:i+len(findLines)], "\n")
			if normalize(block) == normalizedFind {
				out = append(out, block)
			}
		}
	}
	return out
}

func indentationFlexibleReplacer(content, find string) []string {
	removeIndent := func(text string) string {
		lines := strings.Split(text, "\n")
		nonEmpty := 0
		minIndent := -1
		for _, ln := range lines {
			t := strings.TrimSpace(ln)
			if t == "" {
				continue
			}
			nonEmpty++
			// count leading spaces/tabs
			ind := 0
			for i := 0; i < len(ln); i++ {
				if ln[i] == ' ' || ln[i] == '\t' {
					ind++
				} else {
					break
				}
			}
			if minIndent == -1 || ind < minIndent {
				minIndent = ind
			}
		}
		if nonEmpty == 0 || minIndent <= 0 {
			return text
		}
		for i, ln := range lines {
			t := strings.TrimSpace(ln)
			if t == "" {
				continue
			}
			if len(ln) >= minIndent {
				lines[i] = ln[minIndent:]
			}
		}
		return strings.Join(lines, "\n")
	}

	normalizedFind := removeIndent(find)
	contentLines := strings.Split(content, "\n")
	findLines := strings.Split(find, "\n")
	var out []string
	for i := 0; i <= len(contentLines)-len(findLines); i++ {
		block := strings.Join(contentLines[i:i+len(findLines)], "\n")
		if removeIndent(block) == normalizedFind {
			out = append(out, block)
		}
	}
	return out
}

func escapeNormalizedReplacer(content, find string) []string {
	unescape := func(s string) string {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			ch := s[i]
			if ch == '\\' && i+1 < len(s) {
				i++
				switch s[i] {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				case '\'':
					b.WriteByte('\'')
				case '"':
					b.WriteByte('"')
				case '`':
					b.WriteByte('`')
				case '\\':
					b.WriteByte('\\')
				case '$':
					b.WriteByte('$')
				default:
					// Unknown escape: keep as-is
					b.WriteByte('\\')
					b.WriteByte(s[i])
				}
			} else {
				b.WriteByte(ch)
			}
		}
		return b.String()
	}

	unescapedFind := unescape(find)
	var out []string
	if strings.Contains(content, unescapedFind) {
		out = append(out, unescapedFind)
	}
	lines := strings.Split(content, "\n")
	findLines := strings.Split(unescapedFind, "\n")
	for i := 0; i <= len(lines)-len(findLines); i++ {
		block := strings.Join(lines[i:i+len(findLines)], "\n")
		if unescape(block) == unescapedFind {
			out = append(out, block)
		}
	}
	return out
}

// replace applies the replacers in order and performs either a single unambiguous
// replacement, or if replaceAll is true, replaces all occurrences of the matched search.
// It returns the new content and the number of occurrences replaced (best-effort).
func replace(content, oldString, newString string, replaceAll bool) (string, int, error) {
	if oldString == newString {
		return "", 0, errors.New("old_string and new_string must be different")
	}

	replacers := []replacer{
		simpleReplacer,
		lineTrimmedReplacer,
		blockAnchorReplacer,
		whitespaceNormalizedReplacer,
		indentationFlexibleReplacer,
		escapeNormalizedReplacer,
		// Additional heuristics can be added here if needed
	}

	notFound := true
	for _, r := range replacers {
		for _, search := range r(content, oldString) {
			idx := strings.Index(content, search)
			if idx == -1 {
				continue
			}
			notFound = false
			if replaceAll {
				// Count occurrences for metadata
				count := strings.Count(content, search)
				return strings.ReplaceAll(content, search, newString), count, nil
			}
			lastIdx := strings.LastIndex(content, search)
			if idx != lastIdx {
				// Ambiguous; try the next candidate
				continue
			}
			// Single unique match
			before := content[:idx]
			after := content[idx+len(search):]
			return before + newString + after, 1, nil
		}
	}

	if notFound {
		return "", 0, errors.New("old_string not found in content")
	}
	return "", 0, errors.New("old_string found multiple times and requires more code context to uniquely identify the intended match")
}

// --- Diff utilities ---

// createTwoFilesPatch returns a unified diff of two file contents.
// This is a simple line-based diff producing a single hunk covering the whole file.
func createTwoFilesPatch(src, dst, a, b string) string {
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")
	ops := diffLines(aLines, bLines)

	// hunk header counts
	aCount := len(aLines)
	bCount := len(bLines)

	var out strings.Builder
	out.WriteString("--- ")
	out.WriteString(src)
	out.WriteString("\n")
	out.WriteString("+++ ")
	out.WriteString(dst)
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", 1, aCount, 1, bCount))
	for _, op := range ops {
		switch op.kind {
		case eq:
			for _, s := range op.lines {
				out.WriteString(" ")
				out.WriteString(s)
				out.WriteString("\n")
			}
		case del:
			for _, s := range op.lines {
				out.WriteString("-")
				out.WriteString(s)
				out.WriteString("\n")
			}
		case add:
			for _, s := range op.lines {
				out.WriteString("+")
				out.WriteString(s)
				out.WriteString("\n")
			}
		}
	}
	return out.String()
}

func trimDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var contentLines []string
	for _, line := range lines {
		if line == "" {
			continue
		}
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
		// count leading spaces
		count := 0
		for i := 0; i < len(content); i++ {
			if content[i] == ' ' || content[i] == '\t' {
				count++
			} else {
				break
			}
		}
		if minIndent == -1 || count < minIndent {
			minIndent = count
		}
	}
	if minIndent <= 0 {
		return diff
	}
	for i, line := range lines {
		if line == "" {
			continue
		}
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

// Simple line-diff using LCS to produce operations
type opKind int

const (
	eq opKind = iota
	del
	add
)

type op struct {
	kind  opKind
	lines []string
}

func diffLines(a, b []string) []op {
	// LCS table
	na, nb := len(a), len(b)
	lcs := make([][]int, na+1)
	for i := range lcs {
		lcs[i] = make([]int, nb+1)
	}
	for i := na - 1; i >= 0; i-- {
		for j := nb - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				if lcs[i+1][j] >= lcs[i][j+1] {
					lcs[i][j] = lcs[i+1][j]
				} else {
					lcs[i][j] = lcs[i][j+1]
				}
			}
		}
	}

	i, j := 0, 0
	var ops []op
	appendOp := func(k opKind, s string) {
		if len(ops) > 0 && ops[len(ops)-1].kind == k {
			ops[len(ops)-1].lines = append(ops[len(ops)-1].lines, s)
		} else {
			ops = append(ops, op{kind: k, lines: []string{s}})
		}
	}
	for i < na && j < nb {
		if a[i] == b[j] {
			appendOp(eq, a[i])
			i++
			j++
		} else if lcs[i+1][j] >= lcs[i][j+1] {
			appendOp(del, a[i])
			i++
		} else {
			appendOp(add, b[j])
			j++
		}
	}
	for i < na {
		appendOp(del, a[i])
		i++
	}
	for j < nb {
		appendOp(add, b[j])
		j++
	}
	return ops
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
