# nix2container

**warning: nix2container is in early development stages and interfaces are not stable**

nix2container provides an efficient container development workflow
with images built by Nix: it doesn't write tarballs to the Nix store
and allows to skip already pushed layers (without having to rebuild
them).

This is based on ideas developped in [this blog
post](https://lewo.abesis.fr/posts/nix-build-container-image/).


## Getting started

```nix
{
  inputs.nix2container.url = "github:nlewo/nix2container";

  outputs = { self, nixpkgs, nix2container }: let
    pkgs = import nixpkgs { system = "x86_64-linux"; };
    nix2containerPkgs = nix2container.packages.x86_64-linux;
  in {
    packages.x86_64-linux.hello = nix2containerPkgs.nix2container.buildImage {
      name = "hello";
      config = {
        entrypoint = ["${pkgs.hello}/bin/hello"];
      };
    };
  };
}
```

This image can then be loaded into Docker with

```
$ nix run .#hello.copyToDockerDaemon
$ docker run hello:latest
Hello, world!
```


## More Examples

To load and run the `bash` example image into Podman:

```
$ nix run github:nlewo/nix2container#examples.bash.copyToPodman
$ podman run -it bash
```

- [`bash`](./examples/bash.nix): Bash in `/bin/`
- [`fromImage`](./examples/from-image.nix): Alpine as base image
- [`nginx`](./examples/nginx.nix)
- [`nonReproducible`](./examples/non-reproducible.nix): with a non reproducible store path :/
- [`openbar`](./examples/openbar.nix): set permissions on files (without root nor VM)
- [`uwsgi`](./examples/uwsgi/default.nix): isolate dependencies in layers
- [`layered`](./examples/layered.nix): build a layered image as described in [this blog post](https://grahamc.com/blog/nix-and-layered-docker-images)


## Functions documentation

### `nix2container.buildImage`

Function arguments are:

- **`name`** (required): the name of the image.

- **`tag`** (defaults to the image output hash): the tag of the image.

- **`config`** (defaults to `{}`): an attribute set describing an image configuration as
    defined in the [OCI image
    specification](https://github.com/opencontainers/image-spec/blob/8b9d41f48198a7d6d0a5c1a12dc2d1f7f47fc97f/specs-go/v1/config.go#L23).

- **`contents`** (defaults to `[]`): a list of store paths to include in
    the layer root directory (store path prefixes
    `/nix/store/hash-path` are removed, to relocate it at the image
    `/`).

- **`fromImage`** (defaults to `none`): an image that is used as base
    image of this image.

- **`maxLayers`** (defaults to `1`): the maximum number of layers to
    create. This is based on the store path "popularity" as described
    in this [blog
    post](https://grahamc.com/blog/nix-and-layered-docker-images). Note
    this is applied on the image layers and not on layers added with
    the `buildImage.layers` attribute.

- **`perms`** (defaults to `[]`): a list of file permisssions which are
    set when the tar layer is created: these permissions are not
    written to the Nix store.

    Each element of this permission list is a dict such as
    ```
    { path = "a store path";
      regex = ".*";
      mode = "0664";
    }
    ```
    The mode is applied on a specific path. In this path subtree,
    the mode is then applied on all files matching the regex.

- **`initializeNixDatabase`** (defaults to `false`): to initialize the
    Nix database with all store paths added into the image. Note this
    is only useful to run nix commands from the image, for instance to
    build an image used by a CI to run Nix builds.

- **`layers`** (defaults to `[]`): a list of layers built with the
    buildLayer function: if a store path in deps or contents belongs
    to one of these layers, this store path is skipped. This is pretty
    useful to isolate store paths that are often updated from more
    stable store paths, to speed up build and push time.


### `nix2container.pullImage`

Function arguments are:

- **`imageName`** (required): the name of the image to pull.

- **`imageDigest`** (required): the digest of the image to pull.

- **`sha256`** (required): the sha256 of the resulting fixed output derivation.

- **`os`** (defaults to `linux`)

- **`arch`** (defaults to `x86_64`)

- **`tlsVerify`** (defaults to `true`)


### `nix2container.buildLayer`

For most use cases, this function is not required. However, it could be
useful to explicitly isolate some parts of the image in dedicated
layers, for caching (see the "Isolate dependencies in dedicated
layers" section) or non reproducibility (see the `reproducible`
argument) purposes.

Function arguments are:

- **`deps`** (defaults to `[]`): a list of store paths to include in the
    layer.

- **`contents`** (defaults to `[]`): a list of store paths to include in
    the layer root directory (store path prefixes
    `/nix/store/hash-path` are removed, to relocate it at the image
    `/`).

- **`reproducible`** (defaults to `true`): If `false`, the layer tarball
    is stored in the store path. This is useful when the layer
    dependencies are not bit reproducible: it allows to have the layer
    tarball and its hash in the same store path.

- **`maxLayers`** (defaults to `1`): the maximum number of layers to
    create. This is based on the store path "popularity" as described
    in this [blog
    post](https://grahamc.com/blog/nix-and-layered-docker-images). Note
    this is applied on the image layers and not on layers added with
    the `buildLayer.layers` attribute.

- **`perms`** (defaults to `[]`): a list of file permisssions which are
    set when the tar layer is created: these permissions are not
    written to the Nix store.

    Each element of this permission list is a dict such as
    ```
    { path = "a store path";
      regex = ".*";
      mode = "0664";
    }
    ```
    The mode is applied on a specific path. In this path subtree,
    the mode is then applied on all files matching the regex.

- **`layers`** (defaults to `[]`): a list of layers built with the
    `buildLayer` function: if a store path in deps or contents belongs
    to one of these layers, this store path is skipped. This is pretty
    useful to isolate store paths that are often updated from more
    stable store paths, to speed up build and push time.

- **`ignore`** (defaults to `null`): a store path to ignore when
    building the layer. This is mainly useful to ignore the
    configuration file from the container layer.


## Isolate dependencies in dedicated layers

It is possible to isolate application dependencies in a dedicated
layer. This layer is built by its own derivation: if storepaths
composing this layer don't change, the layer is not rebuilt. Moreover,
Skopeo can avoid to push this layer if it has already been pushed.

Let's consider an `application` printing a conversation. This script
depends on `bash` and the `hello` binary. Because most of the changes
concern the script itself, it would be nice to isolate scripts
dependencies in a dedicated layer: when we modify the script, we only
need to rebuild and push the layer containing the script. The layer
containing dependencies won't be rebuilt and pushed.

As shown below, the `buildImage.layers` attribute allows to
explicitly specify a set of dependencies to isolate.

```nix
{ pkgs }:
let
  application = pkgs.writeScript "conversation" ''
    ${pkgs.hello}/bin/hello
    echo "Haaa aa... I'm dying!!!"
  '';
in
pkgs.nix2container.buildImage {
  name = "hello";
  config = {
    entrypoint = ["${pkgs.bash}/bin/bash" application];
  };
  layers = [
    (pkgs.nix2container.buildLayer { deps = [pkgs.bash pkgs.hello]; })
  ];
}
```

This image contains 2 layers: a layer with `bash` and `hello` closures
and a second layer containing the script only.

In real life, the isolated layer can contains a Python environment or
Node modules.


## Quick and dirty benchmarks

The main goal of nix2container is to provide fast rebuild/push
container cycles. In the following, we provide an order of magnitude
of rebuild and repush time, for the [`uwsgi` image](https://github.com/nlewo/nix2container/blob/c6a8d82f1cdd80fabb76e7c1459471e1ea95a080/examples/uwsgi/default.nix).

**warning: this is quick and dirty benchmarks which only provide an order of magnitude**

We build the container and push the container. We then made a small
change in the `hello.py` file to trigger a rebuild and a push.

Method | Rebuild/repush time | Executed command
---|---|---
nix2container.buildImage | ~1.8s | `nix run .#example.uwsgi.copyToRegistry`
dockerTools.streamLayeredImage | ~7.5s | `nix build .#example.uwsgi \| docker load`
dockerTools.buildImage | ~10s | `nix build .#example.uwsgi; skopeo copy docker-archive://./result docker://localhost:5000/uwsgi:latest`

Note we could not compare the same distribution mechanisms because
- Skopeo is not able to skip already loaded layers by the Docker daemon and
- Skopeo failed to push to the registry an image streamed to stdin.


## Run the tests

```
nix run .#tests.all
```

This builds several example images with Nix, loads them with Skopeo,
runs them with Podman, and test output logs.

Not that, unfortunately, these tests are not executed in the Nix
sandbox because it is currently not possible to run a container in the
Nix sandbox.

It is also possible to run a specific test:

```
nix run .#tests.basic
```


## The nix2container Go library

This library is currently used by the Skopeo `nix` transport available
in [this branch](https://github.com/nlewo/image/tree/nix).

For more information, refer to [the Go
documentation](https://pkg.go.dev/github.com/nlewo/nix2container).
