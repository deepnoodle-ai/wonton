# Skills

Agent skills for working with Wonton. Each skill is a directory containing a `SKILL.md` with YAML
frontmatter, plus reference files loaded on demand. The format is portable across agents that
support the SKILL.md convention.

| Skill | Description |
| --- | --- |
| [`wonton`](wonton/) | Building Go CLIs and terminal UIs with the Wonton packages |

## Install

Copy or symlink the skill directory into your agent's skills folder.

**Claude Code** — per-project or per-user:

```bash
mkdir -p .claude/skills && cp -r skills/wonton .claude/skills/       # this project
mkdir -p ~/.claude/skills && cp -r skills/wonton ~/.claude/skills/   # every project
```

**Codex** — same layout under `.codex`:

```bash
mkdir -p .codex/skills && cp -r skills/wonton .codex/skills/
mkdir -p ~/.codex/skills && cp -r skills/wonton ~/.codex/skills/
```

Both agents discover the skill by its frontmatter `description` and load it when a task matches;
you can also invoke it by name.

To track upstream changes instead of copying, symlink a clone of this repository:

```bash
ln -s "$PWD/skills/wonton" ~/.claude/skills/wonton
```

## Conventions

- Every file is Markdown, at most 200 lines, wrapped at 100 columns.
- `SKILL.md` is the entry point: what the module is, how to choose a package, two working
  examples, and the rules that prevent the most common mistakes.
- `references/*.md` hold the depth. They are linked from `SKILL.md` and read only when needed.
- Code samples are checked against the current source. When a sample and the Go source disagree,
  the source wins — please open an issue or a PR.
