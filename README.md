# nix2container

**warning: nix2container is in early development stages and interfaces are not stable**

nix2container provides an efficient container development workflow
with images built by Nix: it doesn't write tarballs to the Nix
store and allows to skip already pushed layers (with having to rebuild
them).

nix2container is
- a Nix library to build container image manifests
- a Go library to produce image configurations and layers from these
  manifests (currently used by Skopeo)

This is based on ideas developped in [this blog
post](https://lewo.abesis.fr/posts/nix-build-container-image/).


## Basic example

```nix
{ pkgs }:
pkgs.nix2container.buildImage {
  name = "basic";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
}
```

This image can be loaded to the Docker deamon with

```shell
$ nix run .#examples.basic.pushToDockerDeamon
Getting image source signatures
Copying blob f4e931379cd5 done
Copying config 88d76532a9 done
Writing manifest to image destination
Storing signatures
Docker image basic:latest have been loaded
```

And run with Docker

```
$ docker run basic:latest
Hello, world!
```


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

As shown below, the `buildImage.isolatedDeps` attribute allows to
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
  isolatedDeps = [
    (pkgs.nix2container.buildLayer { deps = [pkgs.bash pkgs.hello]; })
  ];
}
```

This image contains 2 layers: a layer with `bash` and `hello` closures
and a second layer containing the script only.

In real life, the isolated layer can contains a Python environment or
Node modules.

## Use the Go library

The Go library exposes 
- `NewImageFromFile`: create an `Image` object from a JSON file (created by `nix2container.buildImage`),
- `GetConfigBlob`: generate the config of an `Image`,
- `GetBlob`: get a blob from an `Image`.

This library is currently used by the Skopeo `nix` transport available
in [this branch](https://github.com/nlewo/image/tree/nix).
