# projectr

A minimal CLI tool for freelancers to scaffold project folders with a consistent structure.

## Usage

```
projectr <project-name> [-s file1 file2 ...]
```

### Examples

```bash
# Create a project folder
projectr "34 Branches and borders (Alexs1)"

# Create a project and copy source files into src/
projectr "35 Logo redesign (maria22)" -s brief.pdf logo_v1.ai
```

## Installation

1. Clone the repository or download `main.go`
2. Build the binary:

```bash
go build -o projectr main.go
```

3. Move it to your PATH:

```bash
sudo mv projectr /usr/local/bin/
```

## First Run

On first launch, projectr will ask you to set the path to your orders folder (defaults to `~/Documents/Orders`). If the folder doesn't exist, it will offer to create it. The path is saved to `~/.config/projectr/config`.

```
┌─────────────────────────────────────────┐
│          projectr — first run            │
└─────────────────────────────────────────┘

Enter path to your orders folder
(press Enter to use: /home/user/Documents/Orders):
```

## Project Structure

Each new project gets the following folder layout:

```
34 Branches and borders (Alexs1)/
├── src/          — source files from client
├── drafts/       — work in progress
├── final/        — deliverables for client
├── references/   — reference materials
└── README.md     — auto-generated project info
```

## Configuration

Config file location: `~/.config/projectr/config`

```
OrderPath=/home/user/Documents/Orders
```

You can edit this file directly at any time.

### Flags

| Flag       | Description                              |
|------------|------------------------------------------|
| `-s`       | Source files to copy into `src/`         |
| `--config` | Print path to the config file            |
| `--reset`  | Remove config and re-run setup on next launch |
| `--help`   | Show usage information                   |

## Requirements

- Go 1.18 or later
