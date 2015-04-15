#!/bin/bash

#settings
EXEC="%s"
MOVE="%v"
BIN_DIR=/usr/local/bin

#bash check
if [ ! "$BASH_VERSION" ] ; then
    echo "Please use bash instead" 1>&2
    exit 1
fi

function fail {
	msg=$1
	echo "============"
    echo "Error: $msg" 1>&2
    exit 1
}

#dependency check
if ! which curl > /dev/null; then
	fail "curl is not installed"
fi

#find OS
case `uname -s` in
Darwin) OS="darwin";;
Linux) OS="linux";;
*) echo "unknown os" && exit 1;;
esac
#find ARCH
if uname -m | grep 64 > /dev/null; then
	ARCH="amd64"
else
	ARCH="386"
fi

GH="https://github.com/jpillora/$EXEC"
#releases/latest will 302, inspect Location header, extract version
VERSION=`curl -sI $GH/releases/latest |
		grep Location |
		sed "s~^.*tag\/~~" | tr -d '\n' | tr -d '\r'`


#confirm version
if [ "$VERSION" = "" ]; then
	echo "Latest release not found: $GH"
	exit 1
fi

#download!
DIR="${EXEC}_${VERSION}_${OS}_${ARCH}"
echo "Downloading: $DIR"
URL="$GH/releases/download/$VERSION/$DIR"
case "$OS" in
darwin)
	curl -# -L "$URL.zip" > tmp.zip || fail "download failed"
	unzip -o -qq tmp.zip || fail "unzip failed"
	rm tmp.zip || fail "cleanup failed"
	;;
linux)
	curl -# -L "$URL.tar.gz" | tar zxf - || fail "download failed"
	;;
esac

#move into PATH or cwd
if [[ $MOVE = "true" && -d $BIN_DIR ]]; then
	mv $DIR/$EXEC $BIN_DIR/$EXEC || fail "mv failed"
	chmod +x $BIN_DIR/$EXEC || fail "chmod +x failed"
	echo "Installed at $BIN_DIR/$EXEC"
else
	mv $DIR/$EXEC $EXEC || fail "mv failed"
	chmod +x $EXEC || fail "chmod +x failed"
	echo "Downloaded to $(pwd)/$EXEC"
fi

#done
rm -r $DIR || fail "cleanup failed"
