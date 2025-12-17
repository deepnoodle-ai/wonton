# Wonton Documentation Strategy

This document outlines the documentation strategy for Wonton, designed to serve
both human developers and AI coding agents effectively.

## Vision

Wonton's documentation should be **layered, structured, and accessible**. It uses
progressive disclosure: high-level overviews that fit in LLM context windows,
with links to deeper details. Humans get readable, narrative docs with examples.
AI agents get parseable, structured formats they can consume via prompts or tools.

## Core Principles

### Human-Friendly

- Narrative explanations with clear prose
- Runnable code examples
- Visual aids where helpful (diagrams, tables)
- Progressive complexity: start simple, add depth

### AI-Friendly

- Structured formats (Markdown with consistent headings, lists, tables)
- Indexable and link-based to avoid overwhelming context windows
- Machine-parseable schemas (YAML/JSON for APIs)
- Keyword-rich, factual descriptions

### Progressive Disclosure

- Summaries first, then expandable sections or links
- Enables LLMs to process in stages: load summary, then query specifics
- Keeps individual documents focused and context-window-friendly

### Maintainability

- Generate from code where possible (godoc comments as source of truth)
- Automate updates via CI/CD
- Single source of truth: avoid duplication between code and docs

## Documentation Layers

### Layer 1: Repository Root

**README.md** serves as the entry point for both humans and AI:

- High-level description of Wonton's purpose and philosophy
- Package index with brief descriptions and links
- Quickstart guide for common use cases
- Installation instructions

**llms.txt** follows the llms.txt standard for LLM accessibility:

- Brief repo summary (1-2 paragraphs, optimized for parsing)
- Package index with one-sentence descriptions and links
- Links to key guides and API documentation
- Guidelines for LLMs on how to navigate the docs

The llms.txt file acts as an index, helping AI agents understand what's
available and where to find it without loading everything at once.

**CLAUDE.md** provides project context for Claude Code and similar tools:

- Development commands and workflow
- Package overview table
- Coding conventions specific to this project

### Layer 2: Per-Package Documentation

Each package maintains its own documentation:

**In-Code Documentation (Godoc)**

- Package-level overview comment
- Type and function documentation
- Runnable examples in `_test.go` files
- This is the source of truth for API documentation

**Package README.md**

- 1-paragraph summary (fits small context windows)
- Usage examples with complete, runnable code
- API tables for quick reference
- Links to related packages

**Examples Directory**

- Standalone, runnable programs demonstrating package usage
- Serve as both documentation and integration tests
- Each example should be self-contained and well-commented

### Layer 3: Conceptual Documentation

The `/docs/` directory contains cross-cutting documentation:

- Architecture and design decisions
- Tutorials that span multiple packages
- Integration guides
- This strategy document

## AI Agent Integration

### Structured Formats

- Use tables for API summaries (function, description, inputs, outputs)
- Include YAML/JSON schemas for complex types
- Consistent heading structure for reliable parsing

### Prompt Templates

Consider including ready-to-use prompts in documentation:

```
You are working with the Wonton cli package. Given a command specification,
generate the command handler code following the patterns in the examples.
```

### Skills and Tools

For deeper AI integration, define tool schemas that agents can use:

- Each package can have a corresponding tool definition
- Use standard formats (JSON Schema, OpenAPI-like YAML)
- Enable agents to understand package capabilities programmatically

Example structure:
```yaml
name: parse_cli_args
description: Parses command-line arguments using the cli package
parameters:
  type: object
  properties:
    args:
      type: array
      items:
        type: string
```

## Documentation Standards

### Writing Style

- Active voice, present tense
- Concise but complete
- Code examples should compile and run
- Avoid jargon; explain domain terms

### Structure

- Every package has a README.md
- Every public API has a godoc comment
- Complex features have examples in `/examples/`
- Cross-package concepts go in `/docs/`

### Code Examples

- Must be complete and runnable
- Show common use cases first
- Include error handling
- Demonstrate idiomatic Go patterns

## Implementation Phases

### Phase 1: Foundation

- Establish root README with package index
- Create llms.txt with package summaries
- Ensure all packages have basic godoc comments

### Phase 2: Per-Package Depth

- Add README.md to each package
- Create runnable examples for key packages
- Add API tables to package READMEs

### Phase 3: Conceptual Docs

- Architecture documentation
- Cross-package tutorials
- Integration guides

### Phase 4: AI Enhancements

- Tool/skill definitions for key packages
- Prompt templates for common tasks
- Validate with AI agents (test doc consumption)

## Automation

### From Code

- Generate API documentation from godoc comments
- Use tools like `gomarkdoc` to produce Markdown from Go code
- Keep generated and hand-written docs clearly separated

### CI/CD

- Validate docs build on pull requests
- Auto-update generated docs on merge
- Link checking to catch broken references

### Validation

- Test that examples compile and run
- Use AI agents to verify docs are consumable (no hallucination on queries)
- Aim for summaries under 1k tokens; link to details

## Success Criteria

Documentation is successful when:

1. A new developer can get started with any package in under 5 minutes
2. An AI agent can understand package capabilities from llms.txt + README
3. Common questions are answered without reading source code
4. Examples can be copied, modified, and used directly
5. Docs stay in sync with code through automation

## Related Resources

- [llms.txt standard](https://llmstxt.org/)
- [Go documentation conventions](https://go.dev/doc/comment)
- [pkg.go.dev](https://pkg.go.dev/) for hosted godoc
