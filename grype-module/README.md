# grype-module

A Dagger module for scanning directories for vulnerabilities using [Grype](https://github.com/anchore/grype).

## Features
- Scans any directory for OS/package vulnerabilities
- Returns results in JSON, table, or other Grype-supported formats
- Fully containerized and reproducible
- Easy integration with any Dagger pipeline

## Usage

### Scan a Directory (default: JSON output)
```sh
dagger call scan-directory --directory-arg=. --output-format=json
```

### Output as Table
```sh
dagger call scan-directory --directory-arg=. --output-format=table
```

### Function Signature
```
ScanDirectory(ctx context.Context, directoryArg *dagger.Directory, outputFormat string) (string, error)
```
- `directoryArg`: The directory to scan (mounted in the container)
- `outputFormat`: Grype output format (e.g., `json`, `table`, `cyclonedx`, etc.)

## Integration Example (from another Dagger pipeline)

If this module is in a sibling directory:
```sh
dagger call --mod ../grype-module scan-directory --directory-arg=./some/dir --output-format=json
```

## Requirements
- Dagger CLI
- Internet access (to install Grype in the container)

## License
MIT or Apache-2.0 (add your preferred license) 