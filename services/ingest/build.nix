# https://nixos.org/manual/nixpkgs/stable/#var-stdenv-sourceRoot
# https://nixos.org/manual/nixpkgs/stable/#sec-language-go

{ stdenv, pkgs, buildGoModule, lib, ... }:
let
  hash = "sha256-kOHI8g/EV+kqgIiWhWZfgKc6/F5Jv3XAqqRoxp65L4k=";

  # buildGoModule behave stdenv.mkDerivation but extended
  ingest = buildGoModule rec {
      version = "1.0";
      src = ./../..;
      name = "ingest-${version}";

      vendorHash = hash;

      sourceRoot = "stream-platform/services/ingest";

      prePatch = ''
      ls -f
      '';
  };
in
ingest

