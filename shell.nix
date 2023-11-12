{ pkgs ? import (fetchTarball "channel:nixos-unstable") {} }:

pkgs.mkShell {
  packages = with pkgs; [
    delve
    go_1_21
    gopls
    goreleaser
  ];

  shellHook = ''
    go install github.com/spf13/cobra-cli@latest
  '';
}
