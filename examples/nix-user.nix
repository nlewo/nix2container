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
  '';

  entrypoint = pkgs.writeShellApplication
    {
      name = "entrypoint";
      text = ''
        (nix --extra-experimental-features nix-command config check && ls -la /nix) >out 2>&1 && cat out
      '';
    };
in
nix2container.buildImage {
  name = "nix-user";

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
    regex = "/home/${user}";
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
    ];
  };
}

