find ingestion-operator -type f -print0 | while IFS= read -r -d '' file; do
    temp_file=$(mktemp)
    sed 's#romashorodok\.com#romashorodok.github.io#g'  "$file" > "$temp_file"
    mv "$temp_file" "$file"
done

