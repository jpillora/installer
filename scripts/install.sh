#!/bin/bash

#settings
EXEC="{{ .Program }}"
DLONLY="{{ .DownloadOnly }}"
RELEASE="{{ .Release }}"
BIN_DIR=/usr/local/bin
TMP_DIR="/tmp/jpinstall"

#bash check
if [ ! "$BASH_VERSION" ] ; then
	echo "Please use bash instead" 1>&2
	exit 1
fi

echo "Downloading $EXEC (release $RELEASE)..."

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
*) fail "unknown os: $(uname -s)";;
esac
#find ARCH
if uname -m | grep 64 > /dev/null; then
	ARCH="amd64"
elif uname -m | grep arm > /dev/null; then
	ARCH="arm"
elif uname -m | grep 386 > /dev/null; then
	ARCH="386"
else
	fail "unknown arch: $(uname -m)"
fi

# LOOP ASSETS
#    URL =


GZ="${EXEC}_${OS}_${ARCH}.gz"
URL="$GH/releases/download/$VERSION/$GZ"
curl -sfI $URL > /dev/null

# IF .tar.gz OR .zip -> LEGACY
ISGZ=$?; if [[ $ISGZ == 0 ]]; then
	echo "Downloading $GZ..."
	#gz download!
	mkdir -p $DIR
	curl -# -L $URL | gzip -d - > $DIR/$EXEC || fail "download failed"
else
	#legacy download!
	DIR="${EXEC}_${VERSION}_${OS}_${ARCH}"
	echo "Downloading $DIR..."
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
fi

#move into PATH or cwd
if [[ $DLONLY = "false" && -d $BIN_DIR ]]; then
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
