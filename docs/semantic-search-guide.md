# Semantic Search Guide - `<search>` Command

## Overview

The `<search>` command enables LLMs to perform AI-powered semantic search across all indexed files in a repository. Unlike traditional text search (grep), semantic search understands the *meaning* of your query and finds conceptually related code, even when exact keywords don't match.

## How It Works

When an LLM includes a `<search>` command, the tool:
1. **Generates query embedding** - Converts the search query to a vector using Ollama's nomic-embed-text model
2. **Compares against index** - Calculates cosine similarity between query and all indexed file embeddings
3. **Filters by threshold** - Only returns results above the minimum similarity score
4. **Ranks results** - Sorts by relevance score descending
5. **Returns formatted output** - Provides file paths, scores, and previews

## Basic Syntax

```
<search query terms>
```

**Examples:**
- `<search authentication logic>` - Find auth-related code
- `<search database connection>` - Find DB connection code
- `<search error handling>` - Find error handling patterns
- `<search API endpoints>` - Find route definitions
- `<search configuration parsing>` - Find config-related code

## Requirements

### Ollama Setup

Search requires [Ollama](https://ollama.com) with the nomic-embed-text model:

```bash
# Install Ollama (Linux)
curl -fsSL https://ollama.com/install.sh | sh

# Install Ollama (macOS)
brew install ollama

# Pull the embedding model
ollama pull nomic-embed-text

# Verify Ollama is running
ollama list
```

### Verify Installation

```bash
# Check Ollama is running
curl http://localhost:11434/api/tags

# Test embedding generation
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "test query"
}'
```

## Building the Search Index

### Initial Index Build

```bash
# Build complete index from scratch
./llm-runtime --reindex
```

This will:
1. Walk through all files in the repository
2. Filter by allowed extensions and excluded paths
3. Generate embeddings using Ollama
4. Store embeddings in SQLite database

### Index Location

The search index is stored at `./embeddings.db` by default (configurable in `llm-runtime.config.yaml`).

## Configuration

In `llm-runtime.config.yaml`:

```
commands:
  search:
    enabled: true
    vector_db_path: "./embeddings.db"
    max_results: 10
    min_similarity_score: 0.3
    max_preview_length: 100
    ollama_url: "http://localhost:11434"
    index_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".ts"
      - ".md"
      - ".txt"
      - ".yaml"
      - ".json"
      - ".rs"
      - ".java"
      - ".c"
      - ".cpp"
    max_file_size: 1048576  # 1MB
```

## Use Cases

### Finding Related Code

```
I need to understand how authentication works in this project.

<search authentication middleware>

Let me also find where tokens are validated:

<search JWT token validation>
```

### Discovering Patterns

```
I want to see how error handling is done across the codebase:

<search error handling patterns>

Now let me find logging implementations:

<search logging and debug output>
```

### Locating Configuration

```
Where is the database configured?

<search database configuration connection string>

And where are environment variables loaded?

<search environment variables loading>
```

### Finding Similar Functions

```
I see there's a user service. Are there similar services?

<search service implementation pattern>

Let me find all the handlers:

<search HTTP request handlers>
```

## Output Format

### Successful Search

```
=== SEARCH: authentication logic ===
=== SEARCH RESULTS ===
1. src/auth/middleware.go (score: 78.50)
   Preview: "// AuthMiddleware validates JWT tokens and extracts user..."

2. src/handlers/login.go (score: 72.30)
   Preview: "// LoginHandler processes user authentication requests..."

3. internal/security/jwt.go (score: 68.90)
   Preview: "// Package security provides JWT token generation and..."

[Showing top 10 results]
=== END SEARCH ===
```

### No Results

```
=== SEARCH: nonexistent feature ===
=== SEARCH RESULTS ===
No files found matching query.
Try broader search terms or check if files are indexed.
=== END SEARCH ===
```

## Best Practices for LLMs

### Start Broad, Then Narrow

```
First, let me get an overview of the authentication system:

<search authentication>

Now let me focus on the specific token handling:

<search JWT token expiration refresh>
```

### Combine with File Reading

```
Let me find files related to database operations:

<search database queries>

Based on the results, I'll examine the top match:

<open src/database/queries.go>
```

### Use Domain-Specific Terms

```
# Good - uses specific terms
<search GraphQL resolver mutations>

# Less effective - too generic  
<search data changes>
```

### Search Before Writing

```
Before creating a new utility, let me check if something similar exists:

<search string manipulation utilities>

Good, nothing similar exists. I'll create a new file:

<write src/utils/strings.go>
// New string utilities
</write>
```

## Troubleshooting

### Search Returns No Results

1. **Check if index exists**:
   ```bash
   ls -la embeddings.db
   ```

2. **Rebuild index**:
   ```bash
   ./llm-runtime --reindex
   ```

3. **Verify Ollama is running**:
   ```bash
   ollama list
   ```

4. **Try broader search terms**

### Ollama Connection Issues

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if needed
ollama serve

# Or as a service (Linux)
sudo systemctl start ollama
```

### Index Out of Date

**When to reindex**:
- After adding many new files
- After major code changes
- After changing `index_extensions` in config
- If search results seem stale

**Reindex commands**:
```bash
# Full rebuild (recommended for major changes)
./llm-runtime --reindex

# Incremental update (faster, only new/modified files)
./llm-runtime search-update

# Check index status
./llm-runtime search-status

# Validate index integrity
./llm-runtime search-validate

# Clean up deleted files
./llm-runtime search-cleanup
```

**Note**: Reindexing is safe - it doesn't affect your source code, only the search database.

### How much disk space does the index use?

Roughly 1-3KB per file (depends on file size):
- 1,000 files ≈ 1-3MB
- 10,000 files ≈ 10-30MB
- 100,000 files ≈ 100-300MB

**Note**: The embedding vector is 768 dimensions × 4 bytes = 3KB per file, plus metadata.

**Typical projects**:
- Small project (100-500 files): <2MB
- Medium project (1,000-5,000 files): 3-15MB  
- Large monorepo (10,000+ files): 30MB+

### Embedding Model Not Found

**Symptom**: `model not found` errors

**Solution**:
```bash
# List available models
ollama list

# Pull the embedding model
ollama pull nomic-embed-text

# Verify it's available
ollama list | grep nomic-embed-text
```

### Performance Issues

**Symptom**: Slow indexing or search

**Solutions**:
1. **Reduce indexed files**: Limit `index_extensions` to necessary types
2. **Increase max_file_size**: Skip very large files
3. **Check Ollama**: Ensure Ollama has sufficient resources
4. **First query slowness**: First search loads model (normal)


## Comparison: Semantic Search vs. Traditional Search

| Aspect | `<search>` (Semantic) | `grep` / `find` |
|--------|----------------------|-----------------|
| Query | Natural language | Exact patterns/regex |
| Finds | Conceptually related | Exact matches |
| Example | `<search user auth>` finds "login", "signin", "authenticate" | `grep "auth"` only finds "auth" |
| Setup | Requires Ollama + index | Built-in |
| Speed | Fast after indexing | Fast always |
| Best for | Discovery, understanding | Known patterns |

## Integration with Other Commands

### Search → Open → Analyze

```
<search error handling middleware>

Based on results, examining the top file:

<open src/middleware/errors.go>

This shows the error handling pattern used throughout the project.
```

### Search → Open → Write → Exec

```
<search similar service implementations>

<open src/services/user_service.go>

I'll create a similar service for products:

<write src/services/product_service.go>
// Implementation based on user_service pattern
package services
...
</write>

Now verify it compiles:

<exec go build ./src/services/...>
```

## Best Practices

### Indexing Strategy

1. **Start with core file types**: Begin with `.go`, `.py`, `.js`, `.md`
2. **Add types as needed**: Expand to `.rs`, `.java`, `.cpp` when working with those languages
3. **Exclude generated files**: Don't index `node_modules`, `vendor`, build outputs
4. **Limit file size**: Keep `max_file_size` at 1MB or less for performance

### Search Query Tips

1. **Be specific**: `JWT token validation` > `auth`
2. **Use domain terms**: `GraphQL resolver mutations` > `data changes`
3. **2-4 words optimal**: Better results than single words or long phrases
4. **Try variations**: If no results, rephrase your query
5. **Use natural language**: The model understands concepts, not just keywords

### Performance Optimization
```yaml
# For large codebases (1000+ files)
commands:
  search:
    max_results: 5              # Fewer results = faster
    min_similarity_score: 0.5   # Higher threshold = more relevant
    max_preview_length: 50      # Shorter previews = faster rendering
```

### Incremental Updates

For active development, use incremental updates instead of full reindex:
```bash
# Daily workflow
./llm-runtime search-update   # Only processes new/changed files

# Weekly/monthly
./llm-runtime --reindex       # Full rebuild for consistency
```



## Summary

The `<search>` command transforms how LLMs explore unfamiliar codebases by enabling conceptual discovery rather than requiring exact keyword knowledge. Powered by Ollama's local embedding model, all processing happens on your machine with no external API calls.
