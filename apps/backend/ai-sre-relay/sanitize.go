package main

import (
	"errors"
	"regexp"
	"strings"
)

// The Holmes /api/chat "analysis" field is the backing LLM's final message,
// verbatim. Smaller models leak two failure shapes into it: un-executed
// tool-call markup emitted as text (typically when the agent loop hits its
// iteration cap) and conversational narration around — or instead of — the
// actual analysis. Both shapes have reached filed Jira tickets verbatim, so
// the raw text is treated as untrusted and cleaned here before any output
// path (Jira, Discord, patch generation) sees it.

var (
	errAnalysisEmpty    = errors.New("empty analysis")
	errAnalysisToolCall = errors.New("analysis is tool-call output, not prose")
)

// toolCallMarkup matches hermes-style tool-call blocks
// (<tool_call>…</tool_call>, possibly left unterminated by a truncated
// response), bare <function=…> blocks some models emit without the wrapper, and
// bare <parameter=…> blocks — the argument spans are matched on their own so a
// block whose opening tags were cut off does not leave its argument values
// behind masquerading as prose.
var toolCallMarkup = regexp.MustCompile(`(?s)<tool_call>.*?(?:</tool_call>|$)|<function=.*?(?:</function>|$)|<parameter=.*?(?:</parameter>|$)`)

// orphanToolCallMarkup matches scaffolding tags left with nothing to pair with
// when a block is truncated at its START — the shape toolCallMarkup cannot see,
// because every one of its alternations needs an opening tag. Only the tags are
// removed, never a surrounding span: a stray closing tag must cost the markup
// and not any real analysis emitted beside it.
var orphanToolCallMarkup = regexp.MustCompile(`</?(?:tool_call|function|parameter)(?:=[^>]*)?>`)

// leadingFiller marks first-person narration ("Now I have a complete picture.
// Let me summarize…") that chat models prepend to an answer.
var leadingFiller = regexp.MustCompile(`(?i)\b(let me|i['’]ll|i will|i['’]ve|i have|i['’]m|i am|now i|first,? i|to summarize|here['’]s (a |the |my )?(summary|what i))\b`)

// trailingFiller marks closing chat offers ("Let me know if…").
var trailingFiller = regexp.MustCompile(`(?i)\b(let me know|feel free to|hope (this|that) helps|happy to (help|dig)|if you( would|['’]d)? like)\b`)

var paragraphBreak = regexp.MustCompile(`\r?\n[ \t]*\r?\n`)

// sanitizeAnalysis validates and cleans one raw LLM analysis. It strips
// tool-call markup and leading/trailing conversational filler, and errors when
// nothing substantive remains — the caller retries or fails the investigation
// instead of publishing garbage.
func sanitizeAnalysis(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errAnalysisEmpty
	}

	hadToolCall := toolCallMarkup.MatchString(s)
	s = strings.TrimSpace(toolCallMarkup.ReplaceAllString(s, ""))
	if orphanToolCallMarkup.MatchString(s) {
		hadToolCall = true
		s = strings.TrimSpace(orphanToolCallMarkup.ReplaceAllString(s, ""))
	}
	if isJSONToolCall(s) {
		hadToolCall = true
		s = ""
	}
	if s == "" {
		if hadToolCall {
			return "", errAnalysisToolCall
		}
		return "", errAnalysisEmpty
	}

	paras := paragraphBreak.Split(s, -1)
	for len(paras) > 0 && isFillerParagraph(paras[0], leadingFiller) {
		paras = paras[1:]
	}
	for len(paras) > 0 && isFillerParagraph(paras[len(paras)-1], trailingFiller) {
		paras = paras[:len(paras)-1]
	}

	out := strings.TrimSpace(strings.Join(paras, "\n\n"))
	if out == "" {
		return "", errAnalysisEmpty
	}
	return out, nil
}

// isJSONToolCall reports whether s is a JSON payload shaped like a tool or
// function call (OpenAI-style {"tool_calls": …}, a bare
// {"name": …, "arguments"/"parameters": …}, or an {"invocation": {"tool": …}}
// envelope) rather than prose analysis.
//
// Parseability is deliberately not required. A block cut off mid-object is not
// valid JSON, and gating on json.Valid let exactly those fragments through to
// be published as analysis. Opening like an object and carrying a tool call's
// key names is the signal; analysis prose is markdown and starts with neither.
func isJSONToolCall(s string) bool {
	if s == "" || (s[0] != '{' && s[0] != '[') {
		return false
	}
	if strings.Contains(s, `"tool_calls"`) || strings.Contains(s, `"tool_call"`) ||
		strings.Contains(s, `"invocation"`) {
		return true
	}
	hasCallee := strings.Contains(s, `"name"`) || strings.Contains(s, `"tool"`)
	hasArgs := strings.Contains(s, `"arguments"`) ||
		strings.Contains(s, `"parameters"`) ||
		strings.Contains(s, `"params"`)
	return hasCallee && hasArgs
}

// isFillerParagraph is deliberately conservative: only short plain prose (no
// markdown structure) carrying conversational markers is treated as filler, so
// a substantive one-line summary survives.
func isFillerParagraph(p string, marker *regexp.Regexp) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return true
	}
	if len(p) > 300 {
		return false
	}
	switch c := p[0]; {
	case c == '#' || c == '-' || c == '*' || c == '|' || c == '`' || c == '>':
		return false
	case c >= '0' && c <= '9':
		return false
	}
	return marker.MatchString(p)
}
