// release-metadata validates one release tag and writes GitHub Actions outputs.
package main

import (
	"flag"
	"fmt"
	"os"

	"yseren/internal/releasemeta"
)

func main() {
	tag := flag.String("tag", "", "release tag in vMAJOR.MINOR.PATCH form")
	flag.Parse()

	metadata, err := releasemeta.Parse(*tag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fmt.Printf("tag=%s\n", metadata.Tag)
	fmt.Printf("version=%s\n", metadata.Version)
	fmt.Printf("android_version_code=%d\n", metadata.AndroidVersionCode)
}
