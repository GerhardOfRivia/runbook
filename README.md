# runbook

A terminal UI for running shell commands in context.

The idea is to provide a modern and intuitive way to run and manage shell
commands that you frequently execute. It is especially useful for system
administrators, developers, and anyone who spends a lot of time in the
terminal.

## quick install

```bash
curl -fsSL https://raw.githubusercontent.com/GerhardOfRivia/runbook/refs/heads/main/install.sh | sh
```

## getting started

```bash
./runbook <file_name>.shbn
```

![demo](./demo.gif)

### make a runbook from markdown file

```bash
./runbook --from-md <file_name>.md > <notebook_name>.shbn
```

### convert runbook to markdown file

```bash
./runbook --to-md <file_name>.shbn > <notebook_name>.md
```

### make a shell script from runbook

```bash
./runbook --to-sh <file_name>.shbn > <script_name>.sh
```

## development

```bash
make build
make test
```

### runbook file format

runbook files (.shbn / .psnb) are based on Jupyter Notebook format. See the [official Jupyter Notebook format](https://github.com/jupyter/nbformat) defined with this JSON schema.
