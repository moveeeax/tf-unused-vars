# tf-unused-vars

[![ci](https://github.com/moveeeax/tf-unused-vars/actions/workflows/ci.yml/badge.svg)](https://github.com/moveeeax/tf-unused-vars/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Find declared-but-unused Terraform **variables**, **locals**, and **outputs** in
a module and fail CI so dead configuration never accumulates.

`terraform validate` is happy with a variable nobody wires, a local nobody reads,
or an output nothing consumes. `tf-unused-vars` walks the HCL, resolves every
`var.` / `local.` reference across all files, and reports the declarations that
are never used — with a non-zero exit code you can gate on.

## Install

```console
go install github.com/moveeeax/tf-unused-vars@latest
```

Or clone and `go build`.

## Usage

```console
$ tf-unused-vars scan ./modules/vpc
KIND      NAME           LOCATION
local     legacy_tags    modules/vpc/main.tf:4
variable  instance_type  modules/vpc/variables.tf:11

2 unused declaration(s) found.
```

Scan multiple paths, emit JSON for tooling, and opt into output checking:

```console
$ tf-unused-vars scan --json --strict-outputs .
```

| Flag | Effect |
| --- | --- |
| `--json` | emit findings as a machine-readable JSON array |
| `--strict-outputs` | also flag outputs that nothing in the module references |

Exit status: `0` clean, `1` unused declarations found, `2` usage or parse error.

## How it works

1. Every `*.tf` file under the given paths is parsed with the official
   `hashicorp/hcl/v2` parser (directories are walked recursively; `.terraform`
   is skipped).
2. Declarations are collected from `variable`, `output`, and `locals` blocks.
3. References are collected by asking every expression in the module for its
   `var.<name>` and `local.<name>` traversals — so a variable declared in
   `variables.tf` and used in `main.tf` resolves correctly.
4. Anything declared but never referenced is reported. Variables and locals
   drive the exit code; outputs are report-only unless `--strict-outputs`,
   because a leaf module's outputs are normally consumed by a caller and look
   unused from the inside.

### Waiving a declaration

Keep a declaration you intend to leave in place by marking it:

```hcl
variable "reserved_for_v2" { # tf-unused-vars:ignore
  type = string
}
```

The marker on the declaration line (or the line just above it) waives it.

## In CI

```yaml
- name: Check for dead Terraform config
  run: |
    go install github.com/moveeeax/tf-unused-vars@latest
    tf-unused-vars scan .
```

## License

MIT — see [LICENSE](LICENSE).
