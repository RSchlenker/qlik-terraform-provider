# Contributing to the Qlik Terraform Provider

Thank you for your interest in contributing to the Qlik Terraform Provider! This document provides guidelines and instructions for setting up your development environment, building, testing, and releasing the provider.

## Prerequisites

Before you can build and contribute to the provider, ensure you have the following installed:

### Required
- **Go** 1.23 or later — Download from [golang.org](https://golang.org/dl)
- **Terraform CLI** 1.13 or later — Download from [terraform.io](https://www.terraform.io/downloads.html)
- **GNU Make** — Usually pre-installed on macOS and Linux; on Windows, use [MinGW](https://www.mingw-w64.org/) or [Git Bash](https://gitforwindows.org/)

### For Releases (optional)
- **GPG key** — Required for signing release tags. See [GitHub's GPG documentation](https://docs.github.com/en/authentication/managing-commit-signature-verification) to set one up.

## Building and Installing Locally

### Build the Provider

To build the provider locally, run:

```bash
make build
```

This compiles the provider binary from the Go source code.

### Install the Provider

To install the provider so it can be used by Terraform on your system, run:

```bash
make install
```

This builds the provider and installs it to your Terraform plugins directory (typically `~/.terraform.d/plugins`).

## Running Acceptance Tests

Acceptance tests validate that the provider works correctly against a live Qlik tenant. These tests require valid credentials and are run by specifying the `TF_ACC=1` environment variable.

### Setting Up Environment Variables

Before running acceptance tests, export the following environment variables with your Qlik tenant credentials:

```bash
export QLIK_TENANT_ID="<your-tenant-id>"
export QLIK_REGION="<your-region>"      # e.g., "us", "eu"
export QLIK_API_KEY="<your-api-key>"
```

You can also add these to a `.env` file in the repository root (not committed) and source it before running tests.

### Running the Tests

To run the acceptance tests, use:

```bash
make testacc
```

This runs all acceptance tests under `./internal/provider/` with a timeout of 120 minutes and parallel execution (up to 10 tests in parallel).

For unit tests only (without acceptance tests), run:

```bash
make test
```

## Regenerating the SDK and Documentation

The provider uses OpenAPI specifications to generate Go SDK code, and generates Terraform documentation using the `tfplugindocs` tool. Both are critical for keeping the code and documentation in sync.

### Running Code Generation

To regenerate all SDK code and documentation, run:

```bash
make generate
```

This command:
1. Runs the copyright header generator (via `copywrite`)
2. Formats example Terraform configurations
3. Generates Terraform provider documentation from code annotations
4. Patches the OpenAPI specs (if needed) and generates Go SDK code from the Qlik OpenAPI specifications

### Why Commit Generated Files?

Generated files (in the `internal/provider/` and `docs/` directories) **must be committed** because:
- They ensure that released versions include complete, up-to-date documentation and SDK code
- The CI pipeline checks for uncommitted generated files to catch missing regeneration

If you modify OpenAPI specs or add new Terraform resources/data sources, you **must run `make generate`** and commit the resulting changes.

## Code Quality

Before committing, ensure your code passes linting and formatting checks:

```bash
make fmt lint
```

The `fmt` target formats all Go code with `gofmt`. The `lint` target runs `golangci-lint`, which enforces code quality standards.

## Making a Release

Releases are published via GitHub Actions using [GoReleaser](https://goreleaser.com/).

### Release Process

1. **Update the changelog** — Add an entry to `CHANGELOG.md` describing the new features, fixes, and improvements.

2. **Create a release tag** — Push a tag to the repository:

   ```bash
   git tag v0.x.y
   git push origin v0.x.y
   ```

   Replace `v0.x.y` with the new version number (e.g., `v0.2.0`).

3. **Sign the tag (recommended)** — If you have a GPG key configured, use it to sign the tag:

   ```bash
   git tag -s v0.x.y -m "Release v0.x.y"
   git push origin v0.x.y
   ```

4. **GitHub Actions takes over** — Once the tag is pushed, the `.github/workflows/release.yml` workflow automatically:
   - Builds the provider for multiple platforms
   - Generates checksums
   - Creates a GitHub Release with binaries
   - Publishes to the Terraform Registry (if configured)

### Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** — Incompatible API changes
- **MINOR** — New features (backward-compatible)
- **PATCH** — Bug fixes (backward-compatible)

For example: `v0.1.0` (first minor release), `v0.1.1` (first patch), `v1.0.0` (first major release).

## Development Tips

### Hot Reloading During Development

After running `make install`, you can edit the Go code and re-run `make install` to rebuild and reinstall without restarting `terraform apply`.

### Running a Subset of Tests

To run only tests for a specific resource or data source:

```bash
go test -v -cover ./internal/provider/ -run TestAccQlikApp
```

Replace `TestAccQlikApp` with the test pattern you want to run.

### Debugging

To run tests with verbose logging:

```bash
TF_LOG=DEBUG TF_ACC=1 go test -v ./internal/provider/ -run <test-pattern>
```

## Repository Structure

- `internal/provider/` — Provider implementation (resources, data sources, schema)
- `docs/` — Terraform provider documentation (generated by `tfplugindocs`)
- `examples/` — Example Terraform configurations
- `tools/` — Build tools and code generation scripts
- `specs/` — OpenAPI specification files for the Qlik API
- `.github/workflows/` — CI/CD pipeline definitions

## Getting Help

- **Issues** — Open a GitHub issue for bugs or feature requests
- **Discussions** — Use GitHub Discussions for questions or ideas
- **Documentation** — Check the `docs/` directory and README.md for more information

Thank you for contributing!
