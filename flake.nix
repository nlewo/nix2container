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
        in !(p == "flake.nix" || p == "flake.lock");
      };
      vendorSha256 = "sha256-gBme4IheJ/cJCRwRH3pnZlU7LKePD2eo7kiZldqQikY=";
    };

    buildLayer = {
      # A list of store path
      contents,
      # A store path to exclude. This is mainly useful to exclude the
      # configuration file from the container layer.
      exclude ? null,
      # A list of layers containing dependencies: if a store path of the
      # currently built layer already belongs to a dependency layer,
      # this store path is skipped
      isolatedDeps ? []
    }:
    pkgs.runCommand "layer.json" {} ''
      echo ${containers-image-nix}/bin/containers-image-nix layer ${pkgs.closureInfo {rootPaths = contents;}}/store-paths
      mkdir $out
      ${containers-image-nix}/bin/containers-image-nix layers-from-non-reproducible-storepaths \
        ${pkgs.closureInfo {rootPaths = contents;}}/store-paths \
        ${pkgs.lib.concatMapStringsSep " "  (l: l + "/layer.json") isolatedDeps} \
        --tar-directory $out \
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
        configLayer = buildLayer {
          contents = configFile;
          exclude = configFile;
          isolatedDeps = isolatedDeps;
        };
        layerPaths = pkgs.lib.concatMapStringsSep " " (l: l + "/layer.json") ([configLayer] ++ isolatedDeps);
      in
      pkgs.runCommand "image.json" {} ''
        echo ${containers-image-nix}/bin/containers-image-nix image ${configFile} ${layerPaths}
        ${containers-image-nix}/bin/containers-image-nix image ${configFile} ${layerPaths} > $out
      '';
    examples = {
      hello = import ./examples/hello.nix { inherit pkgs buildImage buildLayer; };
      nginx = import ./examples/nginx.nix { inherit pkgs buildImage buildLayer; };
      };
  in
  {
    packages.x86_64-linux = {
      inherit containers-image-nix examples;
    };
  };
}
