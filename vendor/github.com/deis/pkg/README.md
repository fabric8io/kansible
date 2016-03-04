# Deis Pkg

[![Build Status](https://travis-ci.org/deis/pkg.svg?branch=master)](https://travis-ci.org/deis/pkg)

The Deis Pkg project contains shared Go libraries that are used by
several Deis projects.

## Usage

Add this project to your `vendor/` directory using Godeps or
[glide](https://github.com/Masterminds/glide):

```
$ glide get --import github.com/deis/pkg
```

(The `--import` flag will get any additional dependencies.)
