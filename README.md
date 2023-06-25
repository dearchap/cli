# Welcome to urfave/cli

[![Run Tests](https://github.com/urfave/cli/actions/workflows/cli.yml/badge.svg)](https://github.com/urfave/cli/actions/workflows/cli.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/urfave/cli/v3.svg)](https://pkg.go.dev/github.com/urfave/cli/v3)
[![Go Report Card](https://goreportcard.com/badge/github.com/urfave/cli/v3)](https://goreportcard.com/report/github.com/urfave/cli/v3)
[![codecov](https://codecov.io/gh/urfave/cli/branch/main/graph/badge.svg?token=t9YGWLh05g)](https://codecov.io/gh/urfave/cli)
<a href="https://discord.gg/UKz76VA6"><img src="https://assets-global.website-files.com/6257adef93867e50d84d30e2/636e0b5061df29d55a92d945_full_logo_blurple_RGB.svg" alt="Discord" width="100" height="20"/> </a>


urfave/cli is a **declarative**, simple, fast, and fun package for building command line tools in Go featuring:

- commands and subcommands with alias and prefix match support
- flexible and permissive help system
- dynamic shell completion for `bash`, `zsh`, `fish`, and `powershell`
- `man` and markdown format documentation generation
- input flags for simple types, slices of simple types, time, duration, and others
- compound short flag support (`-a` `-b` `-c` :arrow_right: `-abc`)
- input lookup from:
    - environment variables
    - plain text files
    - [structured file formats supported via the `urfave/cli-altsrc` package](https://github.com/urfave/cli-altsrc)

## Documentation

More documentation is available in [`./docs`](./docs) or the hosted documentation site published from the latest release
at <https://cli.urfave.org>.

## Q&amp;A

Please check the [Q&amp;A discussions](https://github.com/urfave/cli/discussions/categories/q-a) or [ask a new
question](https://github.com/urfave/cli/discussions/new?category=q-a).

## License

See [`LICENSE`](./LICENSE)
