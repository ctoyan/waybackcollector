#!/usr/bin/env bash

VERSION=$1
OUTPUT_DIR="./release"
TAG="v${VERSION}"
USER="ctoyan"
REPO="waybackcollector"

if [[ -z "$VERSION" || -z "$OUTPUT_DIR" ]]; then
  echo "usage: $0 <VERSION> <OUTPUT_DIR>"
  exit 1
fi

mkdir -p $OUTPUT_DIR

# Check if tag exists
git fetch --tags
git tag | grep "^${TAG}$"

if [ $? -ne 0 ]; then
    github-release release \
        --user $USER \
        --repo $REPO \
        --tag $TAG \
        --name "$REPO $TAG" \
        --description "$TAG" \
        --pre-release
fi

platforms=("windows/amd64" "windows/386" "darwin/amd64" "darwin/386" "freebsd/amd64" "freebsd/386" "linux/amd64" "linux/386")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    BINARY=$REPO'-'$GOOS'-'$GOARCH'-'$VERSION
    if [ $GOOS = "windows" ]; then
        BINARY+='.exe'
    fi

    env GOOS=$GOOS GOARCH=$GOARCH go build -o $OUTPUT_DIR/$BINARY github.com/$USER/$REPO
    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi

		if [[ "$GOOS" == "windows" ]]; then
				ARCHIVE="$REPO-$GOOS-$GOARCH-$VERSION.zip"
				zip -q $OUTPUT_DIR/$ARCHIVE $OUTPUT_DIR/$BINARY
		else
				ARCHIVE="$REPO-$GOOS-$GOARCH-$VERSION.tgz"
				tar --create --gzip --file=$OUTPUT_DIR/$ARCHIVE $OUTPUT_DIR/$BINARY
		fi

		rm $OUTPUT_DIR/$BINARY

		echo "Uploading ${ARCHIVE}..."
        github-release upload \
            --user $USER \
            --repo $REPO \
            --tag $TAG \
            --name "$ARCHIVE" \
            --file $OUTPUT_DIR/$ARCHIVE

		rm -r $OUTPUT_DIR
done
