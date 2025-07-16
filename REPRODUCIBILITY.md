# Reproducibility

DecoyIM currently only supports reproducible builds on Linux with AMD64. This document describes both how to do this, but
also how to verify the existing signatures. The DecoyIM reproducibility process generates a file called `build_info` that
contains the SHA256 sum of DecoyIM binary. Anyone that generates the same file can then generate a detached armored
signature and make that available for others to verify.


## Generating reproducible binaries

In order to generate reproducible binaries, you need to have docker installed. For some operating systems with SELinux
you also need to mark the coyim source directory as being available from inside of Docker, using this command, where
DIR is the coyim source code directory:

```sh
  chcon -Rt svirt_sandbox_file_t $DIR
```

In order to build DecoyIM reproducibly, you simply do

```sh
  make reproducible-linux-build
```

inside of the DecoyIM directory. This will create a new Docker image and then use it to build DecoyIM. At the end of the
process, it will generate two files:

```sh
  bin/coyim
  bin/build_info
```

If you want to sign the `build_info` file using your default GPG key, you can simply run

```sh
  make sign-reproducible
```

This will generate

```sh
  bin/build_info.0xAAAAAAAAAAAAAAAA.rasc
```

where `0xAAAAAAAAAAAAAAAA` is the long-form key ID of your GPG key.

After that you can mail the file to us manually, or use this command:

```sh
  make send-reproducible-signature
```

which will mail the signed `build_info` file to [security@coy.im](mailto:security@coy.im).


## Verifying reproducible binaries

Each release of DecoyIM will have several signatures for the `build_info` file available. You can of course download and
verify each one of those signatures manually, but we also provide a simple way of verifying it using a small Ruby
script. It can be invoked like this:

```sh
  make check-reproducible-signatures
```

This will download everything necessary for the current tag (so you should first check out the tag you want to verify),
and then verify that the coyim binary match the hashes inside of the `build_info` file, and then verify that each
signature checks out for the `build_info` file.
