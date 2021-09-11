#!/bin/bash

package='dsddns'
outdir='./release'
platforms=('windows/386'
           'linux/amd64' 'linux/arm64' 'linux/arm' 'linux/mips'
           'darwin/amd64' 'darwin/arm64')

mkdir -p "$outdir"
for platform in "${platforms[@]}"; do
    split=(${platform//\// })
    GOOS=${split[0]}
    GOARCH=${split[1]}
    output="$outdir/$package-$GOOS-$GOARCH"
    if [[ $GOOS == 'windows' ]]; then
        output+='.exe'
    fi
    echo "$output"
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$output" .
    if [[ $? != 0 ]]; then
        echo 'Aborting...'
        exit 1
    fi
done
exit 0