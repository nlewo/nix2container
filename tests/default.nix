{ pkgs, examples }:

let
  testScript = {
    image,
    command ? "",
    pattern,
  }: pkgs.writeScriptBin "test-script" ''
    ${image.copyToPodman}/bin/copy-to-podman
    ${pkgs.podman}/bin/podman run ${image.imageName}:${image.imageTag} ${command} | grep '${pattern}'
    ret=$?
    if [ $ret -ne 0 ];
    then
      echo "Error: test failed"
      exit $ret
    else
      echo "Test passed"
    fi
  '';
  tests = {
    hello = testScript {
      image = examples.hello;
      pattern = "dying";
    };
    basic = testScript {
      image = examples.basic;
      pattern = "Hello, world";
    };
    openbar = testScript {
      image = examples.openbar;
      pattern = "^drwxrwxrwx.*tmp$";
    };
    bashVersion = testScript {
      image = examples.bash;
      command = "bash --version";
      pattern = "^GNU bash, version";
    };
    bashLs = testScript {
      image = examples.bash;
      command = "ls";
      pattern = "^bin$";
    };
    fromImage = testScript {
      image = examples.fromImage;
      pattern = "/etc/alpine-release$";
    };
    layered = testScript {
      image = examples.layered;
      pattern = "Hello, world";
    };
    nonReproducible = testScript {
      image = examples.nonReproducible;
      pattern = "A non reproducible image built the";
    };
    nested = testScript {
      image = examples.nested;
      pattern = "Hello, world";
    };
    # The /nix have to be explicitly present in the archive with 755 perms
    nonRegressionIssue12 = pkgs.runCommand "test-script" { buildInputs = [pkgs.jq pkgs.gnutar]; } ''
      set -e
      ${examples.basic.copyTo}/bin/copy-to dir://$PWD/image
      cd $PWD/image
      echo "Checking /nix permission are 755 in the tar archive..."
      cat manifest.json | jq -r '.layers[].digest' | cut -d":" -f2 | xargs tar -tvf | grep "^drwxr-xr-x.*nix$"
      echo Test passed
      # TODO: actually this test doesn't need to be run
      mkdir -p $out/bin
      echo echo Test passed > $out/bin/test-script
      chmod a+x $out/bin/test-script
    '';
  };
  all =
    let scripts = pkgs.lib.concatMapStringsSep "\n" (s: "${s}/bin/test-script") (builtins.attrValues tests);
    in pkgs.writeScriptBin "all-test-scripts" ''
      set -e
      ${scripts}
    '';
in tests // { inherit all; }

