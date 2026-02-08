package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/friedenberg/lux/internal/lsp"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error)

type ToolRegistry struct {
	tools    []Tool
	handlers map[string]ToolHandler
	bridge   *Bridge
}

func NewToolRegistry(bridge *Bridge) *ToolRegistry {
	r := &ToolRegistry{
		handlers: make(map[string]ToolHandler),
		bridge:   bridge,
	}
	r.registerBuiltinTools()
	return r
}

func (r *ToolRegistry) List() []Tool {
	return r.tools
}

func (r *ToolRegistry) Call(ctx context.Context, name string, args json.RawMessage) (*ToolCallResult, error) {
	handler, ok := r.handlers[name]
	if !ok {
		return ErrorResult(fmt.Sprintf("unknown tool: %s", name)), nil
	}
	return handler(ctx, args)
}

func (r *ToolRegistry) register(name, description string, schema json.RawMessage, handler ToolHandler) {
	r.tools = append(r.tools, Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
	})
	r.handlers[name] = handler
}

func (r *ToolRegistry) registerBuiltinTools() {
	r.register("lsp_hover", "Get hover information (type info, documentation) at a position in a file",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleHover)

	r.register("lsp_definition", "Go to definition of a symbol at a position",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleDefinition)

	r.register("lsp_references", "Find all references to a symbol at a position",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"},
				"include_declaration": {"type": "boolean", "description": "Include the declaration in results", "default": true}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleReferences)

	r.register("lsp_completion", "Get code completions at a position",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleCompletion)

	r.register("lsp_format", "Format a document",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"}
			},
			"required": ["uri"]
		}`),
		r.handleFormat)

	r.register("lsp_document_symbols", "Get all symbols (functions, classes, variables) in a document",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"}
			},
			"required": ["uri"]
		}`),
		r.handleDocumentSymbols)

	r.register("lsp_code_action", "Get available code actions (quick fixes, refactorings) for a range",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"start_line": {"type": "integer", "description": "0-indexed start line"},
				"start_character": {"type": "integer", "description": "0-indexed start character"},
				"end_line": {"type": "integer", "description": "0-indexed end line"},
				"end_character": {"type": "integer", "description": "0-indexed end character"}
			},
			"required": ["uri", "start_line", "start_character", "end_line", "end_character"]
		}`),
		r.handleCodeAction)

	r.register("lsp_rename", "Rename a symbol across all files",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"},
				"new_name": {"type": "string", "description": "New name for the symbol"}
			},
			"required": ["uri", "line", "character", "new_name"]
		}`),
		r.handleRename)
}

type positionArgs struct {
	URI       string `json:"uri"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
}

type referencesArgs struct {
	positionArgs
	IncludeDeclaration bool `json:"include_declaration"`
}

type formatArgs struct {
	URI string `json:"uri"`
}

type codeActionArgs struct {
	URI            string `json:"uri"`
	StartLine      int    `json:"start_line"`
	StartCharacter int    `json:"start_character"`
	EndLine        int    `json:"end_line"`
	EndCharacter   int    `json:"end_character"`
}

type renameArgs struct {
	positionArgs
	NewName string `json:"new_name"`
}

func (r *ToolRegistry) handleHover(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a positionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Hover(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character)
}

func (r *ToolRegistry) handleDefinition(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a positionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Definition(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character)
}

func (r *ToolRegistry) handleReferences(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a referencesArgs
	a.IncludeDeclaration = true // default
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.References(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character, a.IncludeDeclaration)
}

func (r *ToolRegistry) handleCompletion(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a positionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Completion(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character)
}

func (r *ToolRegistry) handleFormat(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a formatArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Format(ctx, lsp.DocumentURI(a.URI))
}

func (r *ToolRegistry) handleDocumentSymbols(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a formatArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.DocumentSymbols(ctx, lsp.DocumentURI(a.URI))
}

func (r *ToolRegistry) handleCodeAction(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a codeActionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.CodeAction(ctx, lsp.DocumentURI(a.URI),
		a.StartLine, a.StartCharacter, a.EndLine, a.EndCharacter)
}

func (r *ToolRegistry) handleRename(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a renameArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Rename(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character, a.NewName)
}
