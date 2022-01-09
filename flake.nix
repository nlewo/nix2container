{
  description = "A very basic flake";

  outputs = { self, nixpkgs }:
  let
    pkgs = nixpkgs.legacyPackages.x86_64-linux;

    containers-image-nix = pkgs.buildGoModule rec {
      pname = "container-images-nix";
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
      isolatedDeps ? []
    }: let
      rewrites = pkgs.lib.concatMapStringsSep " " (p: "--rewrite '${p},^${p},'") contents;
      allDeps = deps ++ contents;
    in
    pkgs.runCommand "layer.json" {} ''
      mkdir $out
      ${containers-image-nix}/bin/containers-image-nix layers-from-reproducible-storepaths \
        ${pkgs.closureInfo {rootPaths = allDeps;}}/store-paths \
        ${rewrites} \
        ${pkgs.lib.concatMapStringsSep " "  (l: l + "/layer.json") isolatedDeps} \
        ${pkgs.lib.optionalString (exclude != null) "--exclude ${exclude}"} > $out/layer.json
      '';
  
    buildImage = {
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
      in
      pkgs.runCommand "image.json" {} ''
        echo ${containers-image-nix}/bin/containers-image-nix image ${configFile} ${layerPaths}
        ${containers-image-nix}/bin/containers-image-nix image ${configFile} ${layerPaths} > $out
      '';
    examples = {
      hello = import ./examples/hello.nix { inherit pkgs buildImage buildLayer; };
      nginx = import ./examples/nginx.nix { inherit pkgs buildImage buildLayer; };
      bash = import ./examples/bash.nix { inherit pkgs buildImage; };
      };
  in
  {
    packages.x86_64-linux = {
      inherit containers-image-nix examples;
    };
  };
}
