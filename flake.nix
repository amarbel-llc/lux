{
  description = "Lux: LSP Multiplexer";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.1.0";
      in
      {
        packages = {
          default = self.packages.${system}.lux;
          lux = pkgs.buildGoModule {
            pname = "lux";
            inherit version;
            src = ./.;
            vendorHash = null;

            meta = with pkgs.lib; {
              description = "LSP Multiplexer that routes requests to language servers based on file type";
              homepage = "https://github.com/friedenberg/lux";
              license = licenses.mit;
            };
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            delve
            gum
            just
            shfmt
          ];

          shellHook = ''
            echo "Lux: LSP Multiplexer - dev environment"
          '';
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.lux}/bin/lux";
        };
      }
    );
}
