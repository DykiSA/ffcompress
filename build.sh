#!/bin/bash
PLATFORMS=("windows/amd64" "linux/amd64" "darwin/amd64" "darwin/arm64")

for platform in "${PLATFORMS[@]}"
do
    # Split the platform string into OS and Architecture
    PARTS=(${platform//\// })
    export GOOS=${PARTS[0]}
    export GOARCH=${PARTS[1]}
    
    # Determine the extension for Windows
    EXT=""
    if [ $GOOS = "windows" ]; then
        EXT=".exe"
    fi
    
    OUTPUT_NAME="build/ffcompress-${GOOS}-${GOARCH}${EXT}"
    echo "Building for $GOOS/$GOARCH..."
    
    # Compile
    export CGO_ENABLED=0
    go build -ldflags "-s -w" -o $OUTPUT_NAME
done