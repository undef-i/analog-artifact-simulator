# ntsc-wasm

A comprehensive analog video artifact simulator that accurately reproduces classic video imperfections including dot crawl patterns, signal ringing, chroma noise distortion, color bleeding effects, and authentic VHS-style degradation.

For detailed technical implementation, see [Technical Implementation](docs/technical-implementation.md).

[Demo](https://artifact.noxylva.org/)

## Build

### Basic build (default)

```bash
make
```

### Build with technical documentation

```bash
make build-with-docs
```

## Usage

Serve the files in the `dist` directory via an HTTP server.

## Requirements

- Go 1.21+
- Make
- Pandoc (only required for `build-with-docs`)

## License

[GNU AGPL v3.0](LICENSE)