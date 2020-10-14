#!/bin/sh

UNIX_TIME=`date +%s`
VERSION=`git describe --always --long --dirty`
 
cat <<EOF > version.go
package main

import "time"

var (
	buildTime = time.Unix($UNIX_TIME, 0)
	version = "$VERSION"
)
EOF
