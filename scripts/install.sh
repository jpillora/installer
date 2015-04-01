#!/bin/bash

EXEC="%s"
# EXEC="chisel"

if ! which curl > /dev/null; then
	echo "curl is not installed"
	exit 1
fi

#find OS
case `uname -s` in
Darwin) OS="darwin";;
Linux) OS="linux";;
*) echo "unknown os" && exit 1;;
esac
#find ARCH
if uname -m | grep 64 > /dev/null; then
	ARCH=amd64
else
	ARCH=386
fi

GH="https://github.com/jpillora/$EXEC"
VERSION=`curl -sI $GH/releases/latest | grep Location | sed 's/.*releases\/tag\///'`

if [ "$VERSION" = "" ]; then
	echo "Latest release not found: $GH"
	exit 1
fi

DIR="${EXEC}_${VERSION}_${OS}_${ARCH}"
echo "Downloading: $DIR"
URL="$GH/releases/download/$VERSION/$DIR"
case "$OS" in
darwin)
	curl -# -L "$URL.zip" > tmp.zip 
	unzip -qq tmp.zip
	rm tmp.zip
	;;
linux)
	curl -# -L "$URL.tar.gz" | tar zxvf -
	;;
esac

cp $DIR/$EXEC $EXEC
chmod +x $EXEC
rm -r $DIR
echo "Done"