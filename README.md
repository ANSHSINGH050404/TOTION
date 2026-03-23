# Totion 🧠

A beautiful and minimal Terminal User Interface (TUI) application built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Installation

### npm (recommended)

Install globally from npm:

```bash
npm install -g totion-cli
```

Then run:

```bash
totion
```

### Go install (alternative)

If you prefer building from source with Go:

```bash
go install github.com/ANSHSINGH020404/TOTION@latest
```

## Usage

Run the CLI from your terminal:

```bash
totion
```

## Features
- **Modern TUI**: Built using Charm's incredible Lip Gloss and Bubble Tea libraries.
- **Cross-Platform**: Works on Windows, macOS, and Linux.

## Development

To run the project locally during development:

```bash
go run main.go
```

To build a standalone executable:

```bash
go build -o totion .
```

## Publish to npm

This repository includes a Node.js launcher (`bin/totion.cjs`) that runs the correct prebuilt binary from `build/`.

Before publishing:

1. Update version in `package.json`.
2. Login to npm:

   ```bash
   npm login
   ```

3. Publish:

   ```bash
   npm publish --access public
   ```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
