# LLM File Access Tool Documentation

This directory contains comprehensive documentation for llm-runtime.

## Quick Start

1. **New users**: Start with [llm-runtime-overview.md](llm-runtime-overview.md)
2. **Installation**: Follow [installation-guide.md](installation-guide.md)
3. **LLM Integration**: Use [SYSTEM_PROMPT.md](SYSTEM_PROMPT.md)
4. **Configuration**: Check [configuration.md](configuration.md)

## Main Documentation

| Document | Description |
|----------|-------------|
| [llm-runtime-overview.md](llm-runtime-overview.md) | High-level overview of what the tool does |
| [architecture.md](architecture.md) | Technical architecture and design principles |
| [SYSTEM_PROMPT.md](SYSTEM_PROMPT.md) | System prompt for LLM integration |

## Feature Guides

| Guide | Command | Description |
|-------|---------|-------------|
| [file-reading-guide.md](file-reading-guide.md) | `<open>` | Reading files safely |
| [file-writing-guide.md](file-writing-guide.md) | `<write>` | Creating and modifying files |
| [command-execution-guide.md](command-execution-guide.md) | `<exec>` | Running commands in Docker |
| [semantic-search-guide.md](semantic-search-guide.md) | `<search>` | AI-powered code search with Ollama |

## Setup & Configuration

| Document | Description |
|----------|-------------|
| [installation-guide.md](installation-guide.md) | Complete installation for all platforms |
| [configuration.md](configuration.md) | All configuration options |
| [docker-cheatsheet.md](docker-cheatsheet.md) | Docker basics for beginners |

## Reference & Help

| Document | Description |
|----------|-------------|
| [quick-reference.md](quick-reference.md) | Command cheat sheet |
| [faq.md](faq.md) | Frequently asked questions |
| [troubleshooting.md](troubleshooting.md) | Common problems and solutions |
| [TODO.md](TODO.md) | Roadmap and future ideas |

## Examples

The [examples/](examples/) directory contains practical tutorials (coming soon):
- Basic exploration workflows
- Language-specific examples
- Common use cases

## Documentation Structure

```
docs/
├── index.md                     # This file
├── llm-runtime-overview.md      # What the tool does
├── architecture.md              # How it works (technical)
├── SYSTEM_PROMPT.md             # LLM integration
│
├── file-reading-guide.md        # <open> command
├── file-writing-guide.md        # <write> command
├── command-execution-guide.md   # <exec> command
├── semantic-search-guide.md     # <search> command
│
├── installation-guide.md        # Setup instructions
├── configuration.md             # Config reference
├── docker-cheatsheet.md         # Docker help
│
├── quick-reference.md           # Cheat sheet
├── faq.md                       # Q&A
├── troubleshooting.md           # Problem solving
├── TODO.md                      # Future plans
│
└── examples/                    # Tutorials (coming soon)
```

## External Links

- **Repository**: [github.com/computerscienceiscool/llm-runtime](https://github.com/computerscienceiscool/llm-runtime)
- **Issues**: [Report bugs](https://github.com/computerscienceiscool/llm-runtime/issues)
