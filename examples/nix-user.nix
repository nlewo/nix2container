{ pkgs, nix2container }:
let
  l = pkgs.lib // builtins;

  user = "user";
  group = "user";
  uid = "1000";
  gid = "1000";

  mkUser = pkgs.runCommand "mkUser" { } ''
    mkdir -p $out/etc/pam.d

    echo "${user}:x:${uid}:${gid}::" > $out/etc/passwd
    echo "${user}:!x:::::::" > $out/etc/shadow

    echo "${group}:x:${gid}:" > $out/etc/group
    echo "${group}:x::" > $out/etc/gshadow

    cat > $out/etc/pam.d/other <<EOF
    account sufficient pam_unix.so
    auth sufficient pam_rootok.so
    password requisite pam_unix.so nullok sha512
    session required pam_unix.so
    EOF

    touch $out/etc/login.defs
    mkdir -p $out/home/${user}

    mkdir -p $out/tmp

    mkdir -p $out/etc/nix
    cat > $out/etc/nix/nix.conf <<EOF
    sandbox = false
    experimental-features = nix-command flakes repl-flake
    EOF
  '';

  entrypoint = pkgs.writeShellApplication
    {
      name = "entrypoint";
      text = ''
        (nix doctor && ls -la /nix) >out 2>&1 && cat out

        # Without arguements, run bash
        if [ $# -eq 0 ]; then
          exec ${pkgs.bash}/bin/bash
        fi

        exec "$@"
      '';
    };
in
nix2container.buildImage {
  name = "nix-user";
  tag = "latest";

  initializeNixDatabase = true;
  nixUid = l.toInt uid;
  nixGid = l.toInt gid;

  copyToRoot = [
    (pkgs.buildEnv {
      name = "root";
      paths = [ pkgs.coreutils pkgs.nix ];
      pathsToLink = "/bin";
    })
    mkUser
  ];

  perms = [{
    path = mkUser;
    regex = "/home/${user}|/tmp|/etc/nix";
    mode = "0744";
    uid = l.toInt uid;
    gid = l.toInt gid;
    uname = user;
    gname = group;
  }];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    User = "user";
    WorkingDir = "/home/user";
    Env = [
      "HOME=/home/user"
      "NIX_PAGER=cat"
      "USER=user"
      "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
    ];
  };
}

