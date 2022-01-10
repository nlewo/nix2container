{ pkgs, buildImage, buildLayer }:
let
    nginxPort = "80";
    nginxConf = pkgs.writeText "nginx.conf" ''
      user nobody nobody;
      daemon off;
      error_log /dev/stdout info;
      pid /dev/null;
      events {}
      http {
        access_log /dev/stdout;
        server {
          listen ${nginxPort};
          index index.html;
          location / {
            root ${nginxWebRoot};
          }
        }
      }
    '';
    nginxWebRoot = pkgs.writeTextDir "index.html" ''
      <html><body><h1>Hello from NGINX</h1></body></html>
    '';
    nginxVar = pkgs.runCommand "nginx-var" {} ''
      mkdir -p $out/var/log/nginx
      mkdir -p $out/var/cache/nginx
    '';
in
  buildImage {
    name = "nginx";
    contents = [
      pkgs.dockerTools.fakeNss
      nginxVar
    ];
    config = {
      Cmd = [ "${pkgs.nginx}/bin/nginx" "-c" nginxConf ];
      ExposedPorts = {
        "${nginxPort}/tcp" = {};
      };
    };
  }
