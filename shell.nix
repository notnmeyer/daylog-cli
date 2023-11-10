{ pkgs ? import (fetchTarball "channel:nixos-23.05") {} }:

pkgs.mkShell {
  packages = with pkgs; [
    delve
    go_1_20
    gopls
    goreleaser
  ];

  shellHook = ''
    go install github.com/spf13/cobra-cli@latest
  '';
}
