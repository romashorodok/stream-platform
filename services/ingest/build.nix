{ stdenv, pkgs,  ... }:

stdenv.mkDerivation rec {
  name = "ingest-${version}";
  version = "1.0";

  src = ./../..;

  buildInputs = with pkgs; [
   go
  ];

  buildPhase = ''
  export HOME=$(pwd)
  cd ./services/ingest

  go build -o ingest ./cmd/ingest/main.go
  	# CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ingest ./cmd/ingest/main.go
  '';

  preInstall = ''
  '';

  installPhase = ''
  mkdir $out
  mv ingest $out/ingest
  '';
}
