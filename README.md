# runbook

An in terminal terminal interface for running commands with context.

The idea is to provide a modern and intuitive way to run and manage commands that you frequently execute. 
It is especially useful for system administrators, developers, and anyone who spends a lot of time in the terminal.

## Getting started

```bash
./runbook <file_name>.shbn
```

## Examples

> shbn files follow the same format as Jupyter Notebook. The official Jupyter Notebook format is defined with this JSON schema.

```json
{
  "cells": [
    {
      "cell_type": "code",
      "execution_count": 1,
      "id": "a1b2c3d4",
      "metadata": {},
      "source": [
        "echo 'Hello, World!'"
      ],
      "outputs": [
        {
          "output_type": "stream",
          "name": "stdout",
          "text": [
            "Hello, World!\n"
          ]
        }
      ]
    },
    {
      "cell_type": "markdown",
      "id": "e5f6g7h8",
      "metadata": {},
      "source": [
        "# This is a Heading\n",
        "This is descriptive text."
      ]
    }
  ],
  "metadata": {},
  "nbformat": 4,
  "nbformat_minor": 5
}
```

## development

```bash
make build
make test
./runbook demo.shbn
```

## Useful resources

- [Jupyter Notebook format](https://github.com/jupyter/nbformat)
