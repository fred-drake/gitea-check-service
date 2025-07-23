{pkgs ? import <nixpkgs> {}}:
pkgs.mkShell {
  buildInputs = with pkgs; [
    git
    alejandra
    delve
    just
    nodejs_22
    uv
    go
    golangci-lint
    govulncheck
    goimports-reviser
  ];

  PROJECT_ROOT = toString ./.;

  shellHook = ''
    git --version
    go version
  '';
}
