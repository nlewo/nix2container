{ pkgs, nix2container, examples }:

let
  testScript = {
    image,
    command ? "",
    grepFlags ? "",
    pattern,
  }: pkgs.writeShellScriptBin "test-script" ''
    ${image.copyToPodman}/bin/copy-to-podman
    ${pkgs.podman}/bin/podman run ${image.imageName}:${image.imageTag} ${command} | ${pkgs.gnugrep}/bin/grep ${grepFlags} '${pattern}'
    ret=$?
    if [ $ret -ne 0 ];
    then
      echo "image list"
      ${pkgs.podman}/bin/podman image list
      echo ""
      echo "Actual output:"
      ${pkgs.podman}/bin/podman run ${image.imageName}:${image.imageTag} ${command}
      echo
      echo "Expected pattern:"
      echo '${pattern}'
      echo
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
    fromImageManifest = testScript {
      image = examples.fromImageManifest;
      pattern = "/etc/alpine-release$";
    };
    configInheritPullImage = pkgs.writeScriptBin "test-script" ''
      ${examples.fromImageConfigInherit.copyToPodman}/bin/copy-to-podman
      
      # Check that CMD is inherited from Alpine
      cmd=$(${pkgs.podman}/bin/podman image inspect ${examples.fromImageConfigInherit.imageName}:${examples.fromImageConfigInherit.imageTag} -f '{{ json .Config.Cmd }}')
      echo "Inherited CMD: $cmd"
      
      if echo "$cmd" | ${pkgs.jq}/bin/jq -e '. == ["/bin/sh"]' > /dev/null; then
        echo "✓ CMD correctly inherited from base Alpine image"
      else
        echo "✗ Expected CMD=['/bin/sh'], got: $cmd"
        exit 1
      fi
      
      # Check that ENV is inherited (Alpine sets PATH)
      env=$(${pkgs.podman}/bin/podman image inspect ${examples.fromImageConfigInherit.imageName}:${examples.fromImageConfigInherit.imageTag} -f '{{ json .Config.Env }}')
      echo "Inherited ENV: $env"
      
      if echo "$env" | ${pkgs.jq}/bin/jq -e 'any(startswith("PATH="))' > /dev/null; then
        echo "✓ ENV correctly inherited from base Alpine image"
      else
        echo "✗ Expected PATH in ENV, got: $env"
        exit 1
      fi
      
      echo "Test passed"
    '';
    configOverridePullImage = pkgs.writeScriptBin "test-script" ''
      ${examples.fromImageConfigOverride.copyToPodman}/bin/copy-to-podman
      
      # Check that CMD is overridden
      cmd=$(${pkgs.podman}/bin/podman image inspect ${examples.fromImageConfigOverride.imageName}:${examples.fromImageConfigOverride.imageTag} -f '{{ index .Config.Cmd 0 }}')
      
      if echo "$cmd" | ${pkgs.gnugrep}/bin/grep -q "bash"; then
        echo "✓ CMD correctly overridden"
      else
        echo "✗ CMD not overridden correctly, got: $cmd"
        exit 1
      fi
      
      # Check that custom ENV was added
      env=$(${pkgs.podman}/bin/podman image inspect ${examples.fromImageConfigOverride.imageName}:${examples.fromImageConfigOverride.imageTag} -f '{{ json .Config.Env }}')
      
      if echo "$env" | ${pkgs.jq}/bin/jq -e 'any(. == "CUSTOM_VAR=test")' > /dev/null; then
        echo "✓ Custom ENV added"
      else
        echo "✗ Expected CUSTOM_VAR=test in ENV"
        exit 1
      fi
      
      # Check that base Alpine PATH is still present (merged, not replaced)
      if echo "$env" | ${pkgs.jq}/bin/jq -e 'any(startswith("PATH="))' > /dev/null; then
        echo "✓ Base PATH ENV preserved during merge"
      else
        echo "✗ Base PATH ENV was lost"
        exit 1
      fi
      
      echo "Test passed"
    '';
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
    ownership = testScript {
      image = examples.ownership;
      pattern = "^-r--r--r-- 1 1001 1001 0 Jan  1  1970 test1.txt";
    };
    # Ensure the Nix database is correctly initialized by querying the
    # closure of the Nix binary.
    nix = testScript {
      image = examples.nix;
      command = "nix-store -qR ${pkgs.nix}";
      pattern = "${pkgs.nix}";
    };
    nix-user = testScript {
      image = examples.nix-user;
      grepFlags = "-Pz";
      pattern = "(?s)\[PASS].*\[PASS].*\[PASS].*drwxr-xr-x \\d+ user user 4096 Jan  1  1970 store";
    };
    # Ensure the Nix database is correctly initialized by querying the
    # closure of the Nix binary.
    # The store path is in a dedicated layer
    nixNested = testScript {
      image = examples.nix;
      command = "nix-store -qR ${pkgs.hello}";
      pattern = "${pkgs.hello}";
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
    nonRegressionIssue45 = testScript {
      image = let
        test = pkgs.runCommand "test" { } ''
          mkdir -p $out/tmp
          touch $out/tmp/test1.txt
          touch $out/tmp/test2.txt
        '';
      in nix2container.buildImage {
        name = "perms";
        config.entrypoint = ["${pkgs.coreutils}/bin/ls" "-l" "/tmp/"];
        copyToRoot = [ test ];
        perms = [
          {
            path = test;
            regex = "test1.txt";
            mode = "0777";
          }
        ];
      };
      # The file test2.txt should not have 777 perms
      pattern = "^-r--r--r-- 1 0 0 0 Jan  1  1970 test2.txt";
    };
    copyToRoot = testScript {
      image = nix2container.buildImage {
        name = "copy-to-root";
        copyToRoot = [ pkgs.hello pkgs.bash ];
        config.entrypoint = ["/bin/bash" "-c" "/bin/hello"];
      };
      pattern = "Hello, world!";
    };
    created = let
      image = examples.created;
      timestamp = "2024-05-13 09:31:10";
    in pkgs.writeScriptBin "test-script" ''
      ${image.copyToPodman}/bin/copy-to-podman
      created=$(${pkgs.podman}/bin/podman image inspect ${image.imageName}:${image.imageTag} -f '{{ .Created }}')
      if echo $created | ${pkgs.gnugrep}/bin/grep '${timestamp}' > /dev/null;
      then
        echo "Test passed"
      else
        echo "Expected Created attribute to contain: ${timestamp}"
        echo ""
        echo "Actual Created attribute: $created"
        echo ""
        echo "Error: test failed"
        exit $ret
      fi
    '';
    # Non regression test for issue https://github.com/nlewo/nix2container/issues/186
    duplicated = let
      image = examples.duplicated;
    in pkgs.writeShellScriptBin "test-script" ''
      ${image.copyToPodman}/bin/copy-to-podman
      length=$(${pkgs.podman}/bin/podman image inspect ${image.imageName}:${image.imageTag} | jq '.[0].RootFS.Layers | length')
      if [[ $length -eq 2 ]]
      then
        echo "Test passed"
      else
        echo "Expected number of layers is 2 while the image contains $length layers"
        echo ""
        echo "Error: test failed"
        exit $ret
      fi
    '';
    metadata = let
      image = examples.metadata;
      expected_created_by = "test created_by";
      expected_author = "test author";
      expected_comment = "test comment";
    in pkgs.writeScriptBin "test-script" ''
      ${image.copyToPodman}/bin/copy-to-podman
      created_by=$(${pkgs.podman}/bin/podman image inspect ${image.imageName}:${image.imageTag} -f '{{ (index .History 0).CreatedBy }}')
      author=$(${pkgs.podman}/bin/podman image inspect ${image.imageName}:${image.imageTag} -f '{{ (index .History 0).Author }}')
      comment=$(${pkgs.podman}/bin/podman image inspect ${image.imageName}:${image.imageTag} -f '{{ (index .History 0).Comment }}')
      if ! echo $created_by | ${pkgs.gnugrep}/bin/grep '${expected_created_by}' > /dev/null;
      then
        echo "Expected created_by attribute to contain: ${expected_created_by}"
        echo ""
        echo "Actual created_by attribute: $created"
        echo ""
        echo "Error: test failed"
        exit 1
      fi
      if ! echo $author | ${pkgs.gnugrep}/bin/grep '${expected_author}' > /dev/null;
      then
        echo "Expected author attribute to contain: ${expected_author}"
        echo ""
        echo "Actual author attribute: $author"
        echo ""
        echo "Error: test failed"
        exit 1
      fi
      if ! echo $comment | ${pkgs.gnugrep}/bin/grep '${expected_comment}' > /dev/null;
      then
        echo "Expected comment attribute to contain: ${expected_comment}"
        echo ""
        echo "Actual comment attribute: $comment"
        echo ""
        echo "Error: test failed"
        exit 1
      fi
      echo "Test passed"
    '';
  } //
  (pkgs.lib.mapAttrs' (name: drv: {
    name = "${name}GetManifest";
    value = pkgs.writeScriptBin "test-script" ''
      set -e
      # Don't pipe directly here, as we don't want to swallow a return code.
      manifest=$(${drv.getManifest}/bin/get-manifest)
      echo "$manifest" | ${pkgs.jq}/bin/jq -e 'has("layers")' > /dev/null
      echo "Test Passed"
    '';
  }) examples.getManifest.images);

  all =
    let scripts =
      pkgs.lib.concatStringsSep
      "\n"
      (pkgs.lib.mapAttrsToList (n: v: "echo Running test '${n}'\n${v}/bin/test-script") tests);
    in pkgs.writeScriptBin "all-test-scripts" ''
      #!${pkgs.runtimeShell}
      set -e
      ${scripts}
    '';
in tests // { inherit all; }
