# https://unix.stackexchange.com/questions/717168/how-to-package-my-software-in-nix-or-write-my-own-package-derivation-for-nixpkgs
# 
{ stdenv, pkgs,  ... }:

stdenv.mkDerivation rec {
  name = "protobuf-${version}";
  version = "1.0";

  src = ./.;

  buildInputs = with pkgs; [
    buf
    nodejs
    protoc-gen-go
    protoc-gen-go-grpc
  ];

  buildPhase = ''
  '';

  installPhase = ''
    mkdir $out
  '';

  shellHook = ''
    cd ./client
    npm install
    cd ..
    buf generate
   '';
}
