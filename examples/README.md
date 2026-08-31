# examples

`demo-module/` is a tiny module that declares one variable it never wires
(`legacy_flag`). Run the linter against it:

```console
$ tf-unused-vars scan ./examples/demo-module
KIND      NAME         LOCATION
variable  legacy_flag  examples/demo-module/main.tf:5

1 unused declaration(s) found.
```

The command exits `1`, so a CI step running it fails the build.
