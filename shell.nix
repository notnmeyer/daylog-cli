{ pkgs ? import (fetchTarball "channel:nixos-23.05") {} }:

pkgs.mkShell {
  packages = with pkgs; [
    delve
    go
    gopls
    goreleaser
  ];

  shellHook = ''
    go install github.com/spf13/cobra-cli@latest
  '';
}
