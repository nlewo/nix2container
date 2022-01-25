{ pkgs, lib, nix2container }:

let
  python = pkgs.python3;
  uwsgi = pkgs.uwsgi.override({
    python3 = python;
    plugins = ["python3"];
  });
  pythonEnv = python.withPackages (p: [p.flask]);
in
nix2container.buildImage {
  name = "uwsgi";
  config = {
    entrypoint = [
      "${uwsgi}/bin/uwsgi"
      "--plugin=python3"
      "--http" ":9090"
      "-H" "${pythonEnv}"
      "--callable" "app"
      "--wsgi-file" ./hello.py
    ];
  };
  # This is to not rebuild/push uwsgi and pythonEnv closures on a
  # hello.py change.
  isolatedDeps = [
    (nix2container.buildLayer {
      deps = [uwsgi pythonEnv];
    })
  ];
}
