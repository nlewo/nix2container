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
  };
  all =
    let scripts = pkgs.lib.concatMapStringsSep "\n" (s: "${s}/bin/test-script") (builtins.attrValues tests);
    in pkgs.writeScriptBin "all-test-scripts" ''
      set -e
      ${scripts}
    '';
in tests // { inherit all; }

