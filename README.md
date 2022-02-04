# sile-fystem

A generic FUSE implementation.

[![Go Reference](https://pkg.go.dev/badge/github.com/JakWai01/sile-fystem.svg)](https://pkg.go.dev/github.com/JakWai01/sile-fystem)

## Overview

`sile-fystem` is a generic FUSE implementation built on top of the `afero` API. Every filesystem supporting the `afero` API can therefore be used as a backend for this FUSE implementation.

A comprehensive reference of FUSE functions can be found [here](https://libfuse.github.io/doxygen/structfuse__operations.html#a1465eb2268cec2bb5ed11cb09bbda42f).

## Installation 

First use `go get` to install the latest version of the package.

```bash
$ go get github.com/JakWai01/sile-fystem
```

Next, include  `sile-fystem` in your application.

```go
import "github.com/JakWai01/sile-fystem"
```

## Usage 

This code is a reference to the `mount_memfs.go` command. It mounts the FUSE using Afero`s MemMapFs as a backend.

```go
logger := logging.NewJSONLogger(5)

serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), "/path/to/mountpoint", "", logger, afero.NewMemMapFs(), false)

cfg := &fuse.MountConfig{
  ReadOnly:                  false,
  DisableDefaultPermissions: false,
}

mfs, err := fuse.Mount("/path/to/mountpoint", serve, cfg)
if err != nil {
  log.Fatalf("Mount: %v", err)
}

if err := mfs.Join(context.Background()); err != nil {
  log.Fatalf("Join %v", err)
}
```

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am "feat: Add something"`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create Pull Request

## License 

sile-fystem (c) 2022 Jakob Waibel and contributors

SPDX-License-Identifier: AGPL-3.0
