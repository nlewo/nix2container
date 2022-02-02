{ pkgs ? import <nixpkgs> { } }:
let
  nix2containerUtil = pkgs.buildGoModule rec {
    pname = "nix2container";
    version = "0.0.1";
    doCheck = false;
    src = pkgs.lib.cleanSourceWith {
      src = ./.;
      filter = path: type:
      let
        p = baseNameOf path;
      in !(
        p == "flake.nix" ||
        p == "flake.lock" ||
        p == "examples" ||
        p == "README.md" ||
        p == "default.nix"
      );
    };
    vendorSha256 = "sha256-o7eE/R8UbuEP0SA+eS0mXb3XeV+gvLfFRDIJ6jvqMuA=";
  };

  skopeo-nix2container = pkgs.skopeo.overrideAttrs (old: {
    preBuild = let
      patch = pkgs.fetchurl {
        url = "https://github.com/nlewo/image/commit/7a612c8a2e20ac7937cae14773cad948c6c7d881.patch";
        sha256 = "sha256-Qeks08itjVuB+fzRxWG7NL27SvZQvD92tAAOO5hpZQs=";
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

  copyToDockerDeamon = image: pkgs.writeScriptBin "copy-to-docker-deamon" ''
    ${skopeo-nix2container}/bin/skopeo --insecure-policy copy nix:${image} docker-daemon:${image.name}:${image.tag}
    ${skopeo-nix2container}/bin/skopeo --insecure-policy inspect docker-daemon:${image.name}:${image.tag}
  '';

  copyToRegistry = image: pkgs.writeScriptBin "copy-to-docker-deamon" ''
    ${skopeo-nix2container}/bin/skopeo --insecure-policy copy nix:${image} docker://${image.name}:${image.tag} $@
    echo Docker image ${image.name}:${image.tag} have copied to registry
  '';

  copyTo = image: pkgs.writeScriptBin "copy-to" ''
    echo Running skopeo --insecure-policy copy nix:${image} $@
    ${skopeo-nix2container}/bin/skopeo --insecure-policy copy nix:${image} $@
  '';

  copyToPodman = image: pkgs.writeShellScriptBin "copy-to-podman" ''
    ${skopeo-nix2container}/bin/skopeo --insecure-policy copy nix:${image} containers-storage:${image.name}:${image.tag}
    ${skopeo-nix2container}/bin/skopeo --insecure-policy inspect containers-storage:${image.name}:${image.tag}
  '';

  # Pull an image from a registry with Skopeo and translate it to a
  # nix2container image.json file.
  # This mainly comes from nixpkgs/build-support/docker/default.nix.
  pullImage =
    let
      fixName = name: builtins.replaceStrings [ "/" ":" ] [ "-" "-" ] name;
    in
    { imageName
      # To find the digest of an image, you can use skopeo:
      # see doc/functions.xml
    , imageDigest
    , sha256
    , os ? "linux"
    , arch ? pkgs.go.GOARCH
    , tlsVerify ? true
    , name ? fixName "docker-image-${imageName}"
    }: let
      dir = pkgs.runCommand name
      {
        inherit imageDigest;
        impureEnvVars = pkgs.lib.fetchers.proxyImpureEnvVars;
        outputHashMode = "recursive";
        outputHashAlgo = "sha256";
        outputHash = sha256;

        nativeBuildInputs = pkgs.lib.singleton pkgs.skopeo;
        SSL_CERT_FILE = "${pkgs.cacert.out}/etc/ssl/certs/ca-bundle.crt";

        sourceURL = "docker://${imageName}@${imageDigest}";
      } ''
      skopeo \
        --insecure-policy \
        --tmpdir=$TMPDIR \
        --override-os ${os} \
        --override-arch ${arch} \
        copy \
        --src-tls-verify=${pkgs.lib.boolToString tlsVerify} \
        "$sourceURL" "dir://$out" \
        | cat  # pipe through cat to force-disable progress bar
      '';
    in pkgs.runCommand "nix2container-${imageName}.json" {} ''
      ${nix2containerUtil}/bin/nix2container image-from-dir $out ${dir}
    '';

  buildLayer = {
    # A list of store paths to include in the layer
    deps ? [],
    # A list of store paths to include in the layer root
    contents ? [],
    # A store path to ignore. This is mainly useful to ignore the
    # configuration file from the container layer.
    ignore ? null,
    # A list of layers containing dependencies: if a store path of the
    # currently built layer already belongs to a dependency layer,
    # this store path is skipped
    isolatedDeps ? [],
    # Store the layer tar in the derivation. This is useful when the
    # layer dependencies are not bit reproducible.
    reproducible ? true,
    # A list of file permisssions which are set when the tar layer is
    # created: these permissions are not written to the Nix store.
    # 
    # Each element of this permission list is a dict such as
    # { path = "a store path";
    #   regex = ".*";
    #   mode = "0664";
    # }
    # The mode is applied on a specific path. In this path subtree,
    # the mode is then applied on all files matching the regex.
    perms ? [],
  }: let
    subcommand = if reproducible
              then "layers-from-reproducible-storepaths"
              else "layers-from-non-reproducible-storepaths";
    rewrites = pkgs.lib.concatMapStringsSep " " (p: "--rewrite '${p},^${p},'") contents;
    permsFile = pkgs.writeText "perms.json" (builtins.toJSON perms);
    permsFlag = pkgs.lib.optionalString (perms != []) "--perms ${permsFile}";
    allDeps = deps ++ contents;
    tarDirectory = pkgs.lib.optionalString (! reproducible) "--tar-directory $out";
  in
  pkgs.runCommand "layers.json" {} ''
    mkdir $out
    ${nix2containerUtil}/bin/nix2container ${subcommand} \
      $out/layers.json \
      ${pkgs.closureInfo {rootPaths = allDeps;}}/store-paths \
      ${rewrites} \
      ${permsFlag} \
      ${tarDirectory} \
      ${pkgs.lib.concatMapStringsSep " "  (l: l + "/layers.json") isolatedDeps} \
      ${pkgs.lib.optionalString (ignore != null) "--ignore ${ignore}"}
    '';

  buildImage = {
    name,
    tag ? "latest",
    # An attribute set describing a container configuration
    config,
    isolatedDeps ? [],
    contents ? [],
    fromImage ? "",
    perms ? [],
  }:
    let
      configFile = pkgs.writeText "config.json" (builtins.toJSON config);
      # This layer contains all config dependencies. We ignore the
      # configFile because it is already part of the image, as a
      # specific blob.
      configDepsLayer = buildLayer {
        inherit contents perms;
        deps = [configFile];
        ignore = configFile;
        isolatedDeps = isolatedDeps;
      };
      fromImageFlag = pkgs.lib.optionalString (fromImage != "") "--from-image ${fromImage}";
      layerPaths = pkgs.lib.concatMapStringsSep " " (l: l + "/layers.json") ([configDepsLayer] ++ isolatedDeps);
      image = pkgs.runCommand "image.json" {} ''
        ${nix2containerUtil}/bin/nix2container image \
        $out \
        ${fromImageFlag} \
        ${configFile} \
        ${layerPaths}
      '';
      namedImage = image // { inherit name tag; };
    in namedImage // {
        copyToDockerDeamon = copyToDockerDeamon namedImage;
        copyToRegistry = copyToRegistry namedImage;
        copyToPodman = copyToPodman namedImage;
        copyTo = copyTo namedImage;
    };
in
{
  inherit nix2containerUtil skopeo-nix2container;
  nix2container = { inherit buildImage buildLayer pullImage; };
}
