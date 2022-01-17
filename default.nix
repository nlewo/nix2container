{ pkgs ? import <nixpkgs> { } }:
let
  nix2containerUtil = pkgs.buildGoModule rec {
    pname = "nix2container";
    version = "0.0.1";
    src = pkgs.lib.cleanSourceWith {
      src = ./.;
      filter = path: type:
      let
        p = baseNameOf path;
      in !(p == "flake.nix" || p == "flake.lock" || p == "examples");
    };
    vendorSha256 = "sha256-gBme4IheJ/cJCRwRH3pnZlU7LKePD2eo7kiZldqQikY=";
  };

  skopeo-nix2container = pkgs.skopeo.overrideAttrs (old: {
    preBuild = let
      patch = pkgs.fetchurl {
        url = "https://github.com/nlewo/image/commit/023556c0d31b155fd73e77ea8d06b7aee87adea8.patch";
        sha256 = "sha256-ygU9jrtxt0KnslWbqjU5fnmwvEfzkIRPjjjil/YiuwQ=";
      };
    in ''
      mkdir -p vendor/github.com/nlewo/nix2container/
      cp -r ${nix2containerUtil.src}/* vendor/github.com/nlewo/nix2container/
      cd vendor/github.com/containers/image/v5
      mkdir nix/
      touch nix/transport.go
      patch -p1 < ${patch}
      cd -
    '';
  });

  pushToDockerDeamon = image: pkgs.writeScriptBin "push-to-docker-deamon" ''
    ${skopeo-nix2container}/bin/skopeo --insecure-policy copy nix:${image} docker-daemon:${image.name}:${image.tag}
    echo Docker image ${image.name}:${image.tag} have been loaded
  '';

  buildLayer = {
    # A list of store paths to include in the layer
    deps,
    # A list of store paths to include in the layer root
    contents ? [],
    # A store path to exclude. This is mainly useful to exclude the
    # configuration file from the container layer.
    exclude ? null,
    # A list of layers containing dependencies: if a store path of the
    # currently built layer already belongs to a dependency layer,
    # this store path is skipped
    isolatedDeps ? [],
    # Store the layer tar in the derivation. This is useful when the
    # layer dependencies are not bit reproducible.
    reproducible ? true
  }: let
    subcommand = if reproducible
              then "layers-from-reproducible-storepaths"
              else "layers-from-non-reproducible-storepaths";
    rewrites = pkgs.lib.concatMapStringsSep " " (p: "--rewrite '${p},^${p},'") contents;
    allDeps = deps ++ contents;
    tarDirectory = pkgs.lib.optionalString (! reproducible) "--tar-directory $out";
  in
  pkgs.runCommand "layer.json" {} ''
    mkdir $out
    ${nix2containerUtil}/bin/nix2container ${subcommand} \
      ${pkgs.closureInfo {rootPaths = allDeps;}}/store-paths \
      ${rewrites} \
      ${tarDirectory} \
      ${pkgs.lib.concatMapStringsSep " "  (l: l + "/layer.json") isolatedDeps} \
      ${pkgs.lib.optionalString (exclude != null) "--exclude ${exclude}"} > $out/layer.json
    '';

  buildImage = {
    name,
    tag ? "latest",
    # An attribute set describing a container configuration
    config,
    isolatedDeps ? [],
    contents ? [],
  }:
    let
      configFile = pkgs.writeText "config.json" (builtins.toJSON config);
      # This layer contains all config dependencies. We exclude the
      # configFile because it is already part of the image, as a
      # specific blob.
      configDepsLayer = buildLayer {
        inherit contents;
        deps = [configFile];
        exclude = configFile;
        isolatedDeps = isolatedDeps;
      };
      layerPaths = pkgs.lib.concatMapStringsSep " " (l: l + "/layer.json") ([configDepsLayer] ++ isolatedDeps);
      image = pkgs.runCommand "image.json" {} ''
        echo ${nix2containerUtil}/bin/nix2container image ${configFile} ${layerPaths}
        ${nix2containerUtil}/bin/nix2container image ${configFile} ${layerPaths} > $out
      '';
      namedImage = image // { inherit name tag; };
    in namedImage // {
        pushToDockerDeamon = pushToDockerDeamon namedImage;
    };
in
{
  inherit nix2containerUtil;
  nix2container = { inherit buildImage buildLayer; };
}
