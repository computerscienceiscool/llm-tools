
# Semantic Search Guide - `<search>` Command

## Overview

The `<search>` command enables LLMs to perform AI-powered semantic search across all indexed files in a repository. Unlike traditional text search (grep), semantic search understands the *meaning* of your query and finds conceptually related code, even when exact keywords don't match.

## How It Works

When an LLM includes a `<search>` command, the tool:
1. **Generates query embedding** - Converts the search query to a 384-dimensional vector using sentence-transformers
2. **Compares against index** - Calculates cosine similarity between query and all indexed file embeddings
3. **Filters by threshold** - Only returns results above the minimum similarity score (default: 0.5)
4. **Ranks results** - Sorts by relevance score descending
5. **Returns formatted output** - Provides file paths, scores, previews, and metadata

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

## Technical Architecture

### Embedding Model

The tool uses the `all-MiniLM-L6-v2` model from sentence-transformers:
- **Dimensions**: 384
- **Speed**: Fast inference
- **Quality**: Good semantic understanding
- **Normalization**: Embeddings are L2-normalized for cosine similarity

### Vector Database

Embeddings are stored in SQLite for simplicity and portability:

```sql
CREATE TABLE embeddings (
    filepath TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    embedding BLOB NOT NULL,
    last_modified INTEGER NOT NULL,
    file_size INTEGER NOT NULL,
    indexed_at INTEGER NOT NULL
);
```

### Similarity Calculation

Cosine similarity is used to compare embeddings:

```
similarity = (A · B) / (||A|| × ||B||)
```

Where A is the query embedding and B is the file embedding. Scores range from 0.0 (unrelated) to 1.0 (identical meaning).

## Requirements

### Python Dependencies

Search requires Python 3.8+ with sentence-transformers:

```bash
# Basic installation
pip install sentence-transformers

# Or with virtual environment (recommended)
python3 -m venv search-env
source search-env/bin/activate
pip install sentence-transformers
```

### Verify Installation

```bash
# Check Python setup
./llm-runtime --check-python-setup

# Or manually test
python3 -c "import sentence_transformers; print('OK')"
```

## Configuration

### Enable Search

In `llm-runtime.config.yaml`:

```yaml
commands:
  search:
    enabled: true
    vector_db_path: "./embeddings.db"
    embedding_model: "all-MiniLM-L6-v2"
    max_results: 10
    min_similarity_score: 0.5
    max_preview_length: 100
    chunk_size: 1000
    python_path: "python3"
    index_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".ts"
      - ".md"
      - ".txt"
      - ".yaml"
      - ".json"
    max_file_size: 1048576  # 1MB
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable/disable search functionality |
| `vector_db_path` | `./embeddings.db` | SQLite database path for embeddings |
| `embedding_model` | `all-MiniLM-L6-v2` | Sentence-transformer model name |
| `max_results` | `10` | Maximum search results to return |
| `min_similarity_score` | `0.5` | Minimum score (0.0-1.0) to include |
| `max_preview_length` | `100` | Characters of preview text per result |
| `chunk_size` | `1000` | Text chunk size for large files |
| `python_path` | `python3` | Path to Python interpreter |
| `index_extensions` | See above | File extensions to index |
| `max_file_size` | `1048576` | Maximum file size to index (bytes) |

## Building the Search Index

### Initial Index Build

```bash
# Build complete index from scratch
./llm-runtime --reindex
```

This will:
1. Walk through all files in the repository
2. Filter by allowed extensions and excluded paths
3. Check if each file is text-based
4. Generate embeddings using Python/sentence-transformers
5. Store embeddings in SQLite database

### Incremental Updates

```bash
# Update only new/modified files
./llm-runtime --search-update
```

### Index Management Commands

```bash
# Show index statistics
./llm-runtime --search-status

# Validate index integrity
./llm-runtime --search-validate

# Remove entries for deleted files
./llm-runtime --search-cleanup

# Full rebuild
./llm-runtime --reindex
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

### Code Review Preparation

```
Before reviewing, let me understand the security measures:

<search input validation sanitization>

<search SQL injection prevention>

<search XSS protection>
```

## Output Format

### Successful Search

```
=== SEARCH: authentication logic ===
=== SEARCH RESULTS (0.45s) ===
1. src/auth/middleware.go (score: 78.50)
   Lines: 156 | Size: 4.2 KB
   Preview: "// AuthMiddleware validates JWT tokens and extracts user..."

2. src/handlers/login.go (score: 72.30)
   Lines: 89 | Size: 2.1 KB
   Preview: "// LoginHandler processes user authentication requests..."

3. internal/security/jwt.go (score: 68.90)
   Lines: 203 | Size: 5.8 KB
   Preview: "// Package security provides JWT token generation and..."

[Showing top 10 results]
=== END SEARCH ===
```

### No Results

```
=== SEARCH: nonexistent feature ===
=== SEARCH RESULTS (0.23s) ===
No files found matching query.
Try broader search terms or check if files are indexed.
=== END SEARCH ===
```

### Relevance Labels

Results include relevance indicators based on score:

| Score Range | Label |
|-------------|-------|
| 90-100% | Excellent |
| 80-89% | Very Good |
| 70-79% | Good |
| 60-69% | Fair |
| 50-59% | Marginal |
| Below 50% | (filtered out) |

## Common Error Types

### SEARCH_DISABLED

```
=== ERROR: SEARCH_DISABLED ===
Message: SEARCH_DISABLED: search feature is not enabled
```

**Cause**: Search not enabled in configuration
**Solution**: Set `commands.search.enabled: true` in config

### SEARCH_INIT_FAILED

```
=== ERROR: SEARCH_INIT_FAILED ===
Message: SEARCH_INIT_FAILED: Python dependencies not available
```

**Cause**: Python or sentence-transformers not installed
**Solution**: 
```bash
pip install sentence-transformers
# Update python_path in config if needed
```

### INDEX_NOT_FOUND

```
=== ERROR: SEARCH_FAILED ===
Message: No files in search index
```

**Cause**: Index hasn't been built yet
**Solution**: Run `./llm-runtime --reindex`

## Performance Considerations

### Index Size

Approximate storage requirements:
- ~1.5 KB per indexed file
- 1,000 files ≈ 1.5 MB index
- 10,000 files ≈ 15 MB index
- 100,000 files ≈ 150 MB index

### Search Speed

- Query embedding generation: ~100-500ms (first query slower due to model loading)
- Similarity calculation: <1ms per file
- Typical search (1000 files): <1 second total

### Optimization Tips

1. **Limit indexed extensions**: Only index relevant file types
2. **Set appropriate file size limits**: Skip very large files
3. **Use incremental updates**: `--search-update` instead of `--reindex`
4. **Adjust result count**: Lower `max_results` if you only need top matches
5. **Tune similarity threshold**: Higher `min_similarity_score` = faster filtering

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

### Verify with Exec

```
Let me find the test files:

<search unit tests for user service>

Now run those specific tests:

<exec go test ./src/services/user_test.go -v>
```

## Troubleshooting

### Search Returns No Results

1. **Check if index exists**:
   ```bash
   ./llm-runtime --search-status
   ```

2. **Verify files are being indexed**:
   - Check `index_extensions` includes your file types
   - Check `excluded_paths` isn't blocking files
   - Verify `max_file_size` isn't too restrictive

3. **Lower similarity threshold**:
   ```yaml
   commands:
     search:
       min_similarity_score: 0.3  # More permissive
   ```

4. **Try broader search terms**

### Search Is Slow

1. **Reduce indexed files**: Narrow `index_extensions`
2. **Lower `max_results`**: Return fewer results
3. **Check Python path**: Ensure it's not loading unnecessary modules

### Index Out of Date

```bash
# Quick update for changed files
./llm-runtime --search-update

# Full rebuild if needed
./llm-runtime --reindex
```

### Python Errors

1. **Check Python version**: Requires 3.8+
   ```bash
   python3 --version
   ```

2. **Verify sentence-transformers**:
   ```bash
   python3 -c "import sentence_transformers; print('OK')"
   ```

3. **Check python_path in config**: Must point to correct interpreter

4. **Virtual environment issues**: Ensure venv is activated or use full path:
   ```yaml
   commands:
     search:
       python_path: "/path/to/venv/bin/python3"
   ```

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

### Search for Test Patterns → Write Tests → Run

```
<search test patterns table driven>

<open src/auth/auth_test.go>

I'll follow this pattern for new tests:

<write src/handlers/api_test.go>
// Table-driven tests following project conventions
...
</write>

<exec go test ./src/handlers/... -v>
```

## Security Considerations

- **Index location**: The embeddings database (`embeddings.db`) should be in `.gitignore`
- **Excluded paths**: Sensitive files (`.env`, `*.key`) are excluded from indexing via `excluded_paths`
- **No content storage**: Only embeddings are stored, not file contents
- **Local processing**: All embedding generation happens locally via Python

## Comparison: Semantic Search vs. Traditional Search

| Aspect | `<search>` (Semantic) | `grep` / `find` |
|--------|----------------------|-----------------|
| Query | Natural language | Exact patterns/regex |
| Finds | Conceptually related | Exact matches |
| Example | `<search user auth>` finds "login", "signin", "authenticate" | `grep "auth"` only finds "auth" |
| Setup | Requires index + Python | Built-in |
| Speed | Fast after indexing | Fast always |
| Best for | Discovery, understanding | Known patterns |

## Summary

The `<search>` command transforms how LLMs explore unfamiliar codebases by enabling conceptual discovery rather than requiring exact keyword knowledge. Combined with `<open>`, `<write>`, and `<exec>`, it provides a complete toolkit for autonomous code exploration, understanding, and modification.
