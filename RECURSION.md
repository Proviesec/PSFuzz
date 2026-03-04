# PSFuzz Recursion Feature 🔄

## Overview

The Recursion feature enables PSFuzz to automatically scan into discovered directories. This makes the scanner significantly more effective at finding hidden content in deep directory structures.

## Features

### 🎯 Smart Recursion
- **Intelligent Status Code Detection**: Only scans directories with interesting status codes (200, 301, 302, 403)
- **Configurable Depth**: Determine how deep the scanner should go
- **Duplicate Prevention**: Already scanned URLs are not scanned again
- **Depth in Output**: Each result in the report includes `depth=N` (0 = base, 1+ = recursive level)

### 🔍 How It Works

1. **Directory Detection**: URLs ending with `/` are recognized as directories
2. **Status Code Check**: Only "interesting" status codes trigger recursion
3. **Automatic Scanning**: Each discovered directory is scanned with the same wordlist
4. **Depth Control**: Maximum depth prevents endless recursion

## Usage

### Command-Line Parameters

```bash
# Basic recursion with depth 3
./psfuzz -u https://example.com -D 3

# With custom status codes for recursion
./psfuzz -u https://example.com -D 3 -rsc "200,301,403"

# Disable smart recursion (scan all directories)
./psfuzz -u https://example.com -D 3 -rs=false

# Greedy recursion: recurse on every match (not only by status)
./psfuzz -u https://example.com -D 3 -recursion-strategy greedy

# Complete example
./psfuzz -u https://example.com -d default -c 20 -D 4 -rsc "200,301,302,403"
```

### Parameters

| Parameter | Short | Description | Default |
|-----------|-------|-------------|---------|
| `--depth` | `-D` | Maximum recursion depth (0=disabled) | 0 |
| `--recursionSmart` | `-rs` | Only interesting status codes | true |
| `--recursionStatusCodes` | `-rsc` | Status codes for recursion | "200,301,302,403" |
| `--recursion-strategy` | – | `default` = recurse only on status (like -rsc); `greedy` = recurse on every match | default |

### Config File

```json
{
    "url": "https://example.com",
    "dirlist": "default",
    "depth": 3,
    "recursionSmart": true,
    "recursionStatusCodes": "200,301,302,403",
    "concurrency": 20
}
```

## Examples

### Example 1: Basic Recursion

```bash
./psfuzz -u https://example.com/api/ -d default -D 2 -c 10
```

**Output (excerpt from report):**  
Each line includes `depth=N`. Base URLs have depth=0, first level depth=1, etc.
```
https://example.com/api/v1/     | 200 | depth=0 | len=... | words=... | ...
https://example.com/api/v1/users/  | 200 | depth=1 | len=1234 | words=89 | ...
https://example.com/api/v1/admin/  | 403 | depth=1 | len=567 | words=23 | ...
```

### Example 2: Only 200 Status Codes

```bash
./psfuzz -u https://example.com -D 3 -rsc "200" -c 15
```

Scans only directories that return status 200.

### Example 3: Aggressive Recursion

```bash
./psfuzz -u https://example.com -D 5 -rs=false -c 20
```

Scans ALL discovered directories regardless of status code (Caution: can take a very long time!).

## Output Format

Recursion results are listed in the same report as base results. Each line includes `depth=N` so you can see the recursion level (0 = base URL level, 1 = first recursive level, etc.). Example excerpt:

```
https://example.com/admin/           | 200 | depth=0 | len=... | ...
https://example.com/admin/users/    | 200 | depth=1 | len=2345 | words=156 | ...
https://example.com/admin/config/   | 403 | depth=1 | len=789 | ...
https://example.com/admin/config/backup/ | 200 | depth=2 | len=1567 | words=98 | ...
```

## Best Practices

### 🎯 Recommended Settings

**Fast Scan:**
```bash
./psfuzz -u https://example.com -D 2 -c 20 -rsc "200,301"
```

**Thorough Scan:**
```bash
./psfuzz -u https://example.com -D 4 -c 15 -rsc "200,301,302,403"
```

**Maximum Scan (Caution!):**
```bash
./psfuzz -u https://example.com -D 5 -c 10 -rs=false -tr 50
```

### ⚠️ Important Notes

1. **Depth Limit**: Higher depths can exponentially increase requests
   - Depth 1: ~1x Wordlist
   - Depth 2: ~2-10x Wordlist
   - Depth 3: ~10-100x Wordlist
   - Depth 4+: Can take a very long time!

2. **Rate Limiting**: Use `-tr` to avoid overloading the server
   ```bash
   ./psfuzz -u https://example.com -D 3 -c 20 -tr 50
   ```

3. **Smart Recursion**: Keep it enabled for more efficient scans
   - Status 200: Successful access
   - Status 301/302: Redirects (often interesting)
   - Status 403: Forbidden (often hidden content)

4. **Concurrency**: Don't set too high with recursion
   - Recommended: 10-20 for Depth 2-3
   - Recommended: 5-10 for Depth 4+

## Performance Tips

### Optimal Configuration

```bash
# Balanced: Fast and thorough
./psfuzz -u https://example.com \
  -d default \
  -D 3 \
  -c 15 \
  -tr 50 \
  -rsc "200,301,403" \
  -fscn 404 \
  -t

# Aggressive: Maximum Speed
./psfuzz -u https://example.com \
  -d fav \
  -D 2 \
  -c 30 \
  -tr 100 \
  -rsc "200,301" \
  -fscn 404,500

# Conservative: Slow and safe
./psfuzz -u https://example.com \
  -d default \
  -D 4 \
  -c 5 \
  -tr 20 \
  -rsc "200,301,302,403"
```

## Troubleshooting

### Problem: Too many requests

**Solution:**
```bash
# Reduce depth and concurrency
./psfuzz -u https://example.com -D 2 -c 5 -tr 20
```

### Problem: No recursion happening

**Check:**
1. Is `-D` > 0 set?
2. Are directories found (URLs with `/`)?
3. Do status codes match `-rsc`?

**Debug:**
```bash
# Show all status codes
./psfuzz -u https://example.com -D 3 -s
```

### Problem: Scan takes too long

**Solution:**
```bash
# Reduce depth or use smaller wordlist
./psfuzz -u https://example.com -D 2 -d fav -c 20
```

## Technical Details

### Algorithm

1. **Initial Scan**: Scan base URL with wordlist
2. **Directory Detection**: Recognize URLs ending with `/`
3. **Status Check**: Check if status code is in `recursionStatusCodes`
4. **Duplicate Check**: Check if URL has already been scanned
5. **Depth Check**: Check if `currentDepth < maxDepth`
6. **Recursive Scan**: Scan discovered directory with same wordlist
7. **Repeat**: Repeat for all discovered directories

### Thread-Safety

- All maps are mutex-protected
- No race conditions
- Safe parallel execution

## Configuration Examples

### Fast Discovery
```json
{
    "url": "https://example.com",
    "depth": 2,
    "recursionSmart": true,
    "recursionStatusCodes": "200,301",
    "concurrency": 20,
    "throttleRate": 50
}
```

### Deep Scan
```json
{
    "url": "https://example.com",
    "depth": 4,
    "recursionSmart": true,
    "recursionStatusCodes": "200,301,302,403",
    "concurrency": 15,
    "throttleRate": 30,
    "filterWrongStatus200": true
}
```

### Maximum Coverage
```json
{
    "url": "https://example.com",
    "depth": 5,
    "recursionSmart": false,
    "concurrency": 10,
    "throttleRate": 20
}
```

## Integration with Other Features

### With Bypass Techniques
```bash
./psfuzz -u https://example.com -D 3 -b -c 15
```

### With Content Filtering
```bash
./psfuzz -u https://example.com -D 3 -fm "admin" -c 15
```

### With Test Length Filtering
```bash
./psfuzz -u https://example.com -D 3 -t -fws -c 15
```

### With Custom Headers
```bash
./psfuzz -u https://example.com -D 3 -rah "Authorization:Bearer token" -c 15
```

## Advanced Usage

### Selective Recursion
Only recurse on specific status codes:
```bash
./psfuzz -u https://example.com -D 3 -rsc "200" -c 20
```

### Recursion strategy: default vs greedy
- **default**: Recursion only when the response status is in `-rsc` (e.g. 200, 301, 302, 403). Same idea as “smart” recursion.
- **greedy**: Recurses on **every** URL that passes the match filter, regardless of status. Use when you want to follow all discovered paths (e.g. 404 pages that might link deeper).

```bash
./psfuzz -u https://example.com -D 3 -recursion-strategy greedy -c 15
```

### Aggressive Mode
Scan everything, no smart detection:
```bash
./psfuzz -u https://example.com -D 4 -rs=false -c 10 -tr 30
```

### Combined with Status Filtering
```bash
./psfuzz -u https://example.com -D 3 -rsc "200,301,403" -fscn 404,500 -c 15
```

## Changelog

### Version 1.0.0
- Recursion with `-D`, `-recursion-strategy default|greedy`, configurable status codes
- Depth in report output (depth=N per result), duplicate prevention, thread-safe

---

Made with ❤️ by [Proviesec](https://proviesec.org/)
