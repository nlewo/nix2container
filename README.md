# nix2container

**warning: nix2container is in early development stages and interfaces are not stable**

nix2container provides an efficient container development workflow
with images built by Nix: it doesn't write tarballs to the Nix
store and allows to skip already pushed layers (with having to rebuild
them).

nix2container is
- a Nix library to build container image manifests
- a binary (`nix2container`) to create JSON files describing layers and images (this
  binary is used by the nix2container Nix library)
- a Go library to produce image configurations and layers from these
  manifests (currently used by Skopeo)

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


## The nix2container Go library

This library is currently used by the Skopeo `nix` transport available
in [this branch](https://github.com/nlewo/image/tree/nix).

For more information, refer to [the Go
documentation](https://pkg.go.dev/github.com/nlewo/nix2container).
