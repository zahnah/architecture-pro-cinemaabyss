#!/usr/bin/env bash
# Exports src/**/*.puml into out/**/*.svg keeping the directory layout. Run from schemas/diagrams.
set -euo pipefail

SRC=src
OUT=out
JAR="${PLANTUML_JAR:-/opt/plantuml.jar}"

mapfile -t dirs < <(find "$SRC" -type f -name '*.puml' -printf '%h\n' | sort -u)
if [[ ${#dirs[@]} -eq 0 ]]; then
  echo "No .puml files under ${SRC}" >&2
  exit 1
fi

exported=0
for dir in "${dirs[@]}"; do
  out_dir="$(pwd)/${OUT}${dir#"$SRC"}"
  mkdir -p "$out_dir"
  mapfile -t files < <(find "$dir" -maxdepth 1 -type f -name '*.puml' | sort)

  java -DRELATIVE_INCLUDE=relative \
    -Dplantuml.include.path="$(pwd)/lib/C4:$(pwd)/lib:$(pwd)/src" \
    -jar "$JAR" -tsvg -o "$out_dir" \
    "${files[@]}"
  exported=$((exported + ${#files[@]}))
done

echo "Exported ${exported} diagram(s) under ${OUT}/"
