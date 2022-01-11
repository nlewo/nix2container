# nix2container

nix2container provides an efficient container development workflow
with images built by Nix: it doesn't write tarballs to the Nix
store and allows to skip already pushed layers (with having to rebuild
them).

nix2container is
- a Nix library to build container image manifests
- a Go library to produce image configurations and layers from these
  manifests (currently used by Skopeo)

## Basic example

```nix
{pkgs, buildImage}:
buildImage {
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
