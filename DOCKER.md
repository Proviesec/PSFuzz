# PSFuzz Docker Guide

This guide explains how to use PSFuzz with Docker for isolated and portable scanning.

## Quick Start

### Build the Image

```bash
docker build -t psfuzz:latest .
```

### Run a Simple Scan

```bash
docker run --rm psfuzz:latest -u https://example.com -w default -c 5
```

## Usage

### Basic Scanning

```bash
# Scan with default wordlist
docker run --rm psfuzz:latest -u https://example.com

# Scan with concurrency
docker run --rm psfuzz:latest -u https://example.com -c 10 -s

# Scan with filters
docker run --rm psfuzz:latest -u https://example.com -fscn 404,403 -c 5
```

### Using Custom Wordlists

Mount your wordlist directory:

```bash
docker run --rm \
  -v $(pwd)/wordlists:/app/wordlists:ro \
  psfuzz:latest \
  -u https://example.com \
  -w /app/wordlists/custom.txt \
  -c 5
```

### Saving Output

Mount an output directory to persist results:

```bash
docker run --rm \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://example.com \
  -w default \
  -c 5 \
  -o /app/output/scan_results
```

### Using Configuration File

Mount your config file (create one from `config.example.json` in the repo if needed):

```bash
docker run --rm \
  -v $(pwd)/config.json:/app/config.json:ro \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -cf /app/config.json
```

## Docker Compose

### Using docker-compose.yml

1. **Edit the configuration:**
   ```bash
   cp docker-compose.yml docker-compose.local.yml
   # Edit docker-compose.local.yml with your settings
   ```

2. **Run the scan:**
   ```bash
   docker-compose -f docker-compose.local.yml up
   ```

3. **View output:**
   ```bash
   cat output/scan.txt
   ```

### Custom Profile

Run the custom profile:

```bash
docker-compose --profile custom up psfuzz-custom
```

## Advanced Usage

### Interactive Shell

Access the container interactively:

```bash
docker run -it --rm \
  --entrypoint /bin/sh \
  psfuzz:latest
```

### Multiple Concurrent Scans

Run multiple scans in parallel:

```bash
# Scan 1
docker run -d --name scan1 \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://site1.com -o /app/output/site1

# Scan 2
docker run -d --name scan2 \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://site2.com -o /app/output/site2

# Check status
docker ps

# View logs
docker logs scan1
```

### Resource Limits

Limit CPU and memory:

```bash
docker run --rm \
  --cpus="2.0" \
  --memory="512m" \
  psfuzz:latest \
  -u https://example.com -c 10
```

### Network Configuration

#### Host Network Mode

For faster performance (Linux only):

```bash
docker run --rm \
  --network host \
  psfuzz:latest \
  -u https://example.com
```

#### Custom DNS

Use custom DNS servers:

```bash
docker run --rm \
  --dns 8.8.8.8 \
  --dns 8.8.4.4 \
  psfuzz:latest \
  -u https://example.com
```

## Examples

### Example 1: Basic Scan with Output

```bash
mkdir -p output
docker run --rm \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://example.com \
  -w default \
  -c 5 \
  -s \
  -o /app/output/example_scan
```

### Example 2: Advanced Scan with Filters

```bash
docker run --rm \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://example.com \
  -w default \
  -c 10 \
  -fscn 404,403 \
  -fws \
  -t \
  -o /app/output/advanced_scan
```

### Example 3: Bypass Techniques

```bash
docker run --rm \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://example.com \
  -w default \
  -c 5 \
  -b \
  -btr \
  -o /app/output/bypass_scan
```

### Example 4: Custom Headers

```bash
docker run --rm \
  psfuzz:latest \
  -u https://api.example.com \
  -w default \
  -rah "Authorization:Bearer token123" \
  -raa "CustomScanner/1.0" \
  -c 5
```

## Image Management

### Build with Specific Version

```bash
docker build -t psfuzz:1.0.0 .
docker tag psfuzz:1.0.0 psfuzz:latest
```

### Image Size

Check image size:

```bash
docker images psfuzz
```

Expected size: ~20-30 MB (Alpine-based)

### Clean Up

Remove images and containers:

```bash
# Remove containers
docker rm $(docker ps -a -q -f ancestor=psfuzz:latest) 2>/dev/null || echo "No containers to remove"

# Or use xargs for safer handling
docker ps -a -q -f ancestor=psfuzz:latest | xargs -r docker rm

# Remove image
docker rmi psfuzz:latest

# Clean build cache
docker builder prune
```

## Directory Structure

Recommended directory structure when using Docker:

```
project/
├── docker-compose.yml
├── config.json
├── wordlists/
│   ├── admin.txt
│   ├── api.txt
│   └── custom.txt
└── output/
    ├── scan1.txt
    └── scan2.txt
```

## Troubleshooting

### Permission Issues

If you encounter permission issues with mounted volumes:

```bash
# Create output directory with correct permissions
mkdir -p output
chmod 755 output  # Or 775 if group write access is needed

# Or run with user mapping
docker run --rm \
  --user $(id -u):$(id -g) \
  -v $(pwd)/output:/app/output \
  psfuzz:latest \
  -u https://example.com -o /app/output/scan
```

### DNS Resolution Issues

If you can't resolve hostnames:

```bash
docker run --rm \
  --dns 8.8.8.8 \
  psfuzz:latest \
  -u https://example.com
```

### Container Won't Start

Check logs:

```bash
docker logs <container-id>
```

### Out of Memory

Increase memory limit:

```bash
docker run --rm \
  --memory="1g" \
  psfuzz:latest \
  -u https://example.com -c 20
```

## Security Considerations

### Running as Non-Root

The Docker image runs as a non-root user (UID 1000) by default for security.

### Read-Only Filesystem

For additional security, mount wordlists as read-only:

```bash
docker run --rm \
  -v $(pwd)/wordlists:/app/wordlists:ro \
  psfuzz:latest \
  -u https://example.com -w /app/wordlists/custom.txt
```

### Network Isolation

Isolate scans in separate networks:

```bash
docker network create scan-network
docker run --rm \
  --network scan-network \
  psfuzz:latest \
  -u https://example.com
```

## Performance Tips

1. **Use Host Network**: On Linux, `--network host` provides better performance
2. **Resource Limits**: Set appropriate CPU/memory limits
3. **Concurrency**: Adjust `-c` flag based on target and resources
4. **Output Volume**: Use local volumes for better I/O performance

## Integration with CI/CD

### GitLab CI Example

```yaml
scan:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker build -t psfuzz:latest .
    - docker run --rm -v $(pwd)/output:/app/output psfuzz:latest -u https://staging.example.com -o /app/output/scan
  artifacts:
    paths:
      - output/
```

### Jenkins Example

```groovy
pipeline {
    agent any
    stages {
        stage('Scan') {
            steps {
                sh 'docker build -t psfuzz:latest .'
                sh 'docker run --rm -v $(pwd)/output:/app/output psfuzz:latest -u https://example.com -o /app/output/scan'
            }
        }
    }
}
```

## Support

For issues or questions:
- GitHub Issues: [https://github.com/Proviesec/PSFuzz/issues](https://github.com/Proviesec/PSFuzz/issues)
- Twitter: [@proviesec](https://twitter.com/proviesec)

---

Made with ❤️ by [Proviesec](https://proviesec.org/)

