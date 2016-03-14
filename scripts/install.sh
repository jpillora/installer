#!/bin/bash
TMP_DIR="/tmp/tmpinstalldir"
function cleanup {
	rm -rf $TMP_DIR > /dev/null
}
function fail {
	cleanup
	msg=$1
	echo "============"
	echo "Error: $msg" 1>&2
	exit 1
}
function install {
	#settings
	USER="{{ .User }}"
	PROG="{{ .Program }}"
	MOVE="{{ .MoveToPath }}"
	RELEASE="{{ .Release }}"
	INSECURE="{{ .Insecure }}"
	OUT_DIR="{{ if .MoveToPath }}/usr/local/bin{{ else }}$(pwd){{ end }}"
	GH="https://github.com"
	#bash check
	if [ ! "$BASH_VERSION" ] ; then
		echo "Please use bash instead" 1>&2
		exit 1
	fi
	if [ ! -d $OUT_DIR ]; then
		fail "output directory missing: $OUT_DIR"
	fi
	#dependency check
	GET=""
	if which curl > /dev/null; then
		GET="curl"
		if [[ $INSECURE = "true" ]]; then GET="$GET --insecure"; fi
		GET="$GET --fail -# -L"
	elif which wget > /dev/null; then
		GET="wget"
		if [[ $INSECURE = "true" ]]; then GET="$GET --no-check-certificate"; fi
		GET="$GET -qO-"
	else
		fail "neither wget/curl are installed"
	fi
	echo "Downloading $PROG (release $RELEASE)..."
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
	#choose from asset list
	URL=""
	FTYPE=""
	case "${OS}_${ARCH}" in{{ range .Assets }}
	"{{ .OS }}_{{ .Arch }}")
		URL="{{ .URL }}"
		FTYPE="{{ .Type }}"
		;;{{end}}
	*) fail "No asset for platform ${OS}-${ARCH}";;
	esac
	#enter tempdir
	mkdir -p $TMP_DIR
	cd $TMP_DIR
	if [[ $FTYPE = ".gz" ]]; then
		which gzip > /dev/null || fail "gzip is not installed"
		#gzipped binary
		NAME="${PROG}_${OS}_${ARCH}.gz"
		GZURL="$GH/releases/download/$RELEASE/$NAME"
		#gz download!
		echo "Downloading $URL"
		bash -c "$GET $URL" | gzip -d - > $PROG || fail "download failed"
	elif [[ $FTYPE = ".tar.gz" ]]; then
		#check if archiver progs installed
		which tar > /dev/null || fail "tar is not installed"
		which gzip > /dev/null || fail "gzip is not installed"
		echo "Downloading $URL"
		bash -c "$GET $URL" | tar zxf - || fail "download failed"
	elif [[ $FTYPE = ".zip" ]]; then
		which unzip > /dev/null || fail "unzip is not installed"
		echo "Downloading $URL"
		bash -c "$GET $URL" > tmp.zip || fail "download failed"
		unzip -o -qq tmp.zip || fail "unzip failed"
		rm tmp.zip || fail "cleanup failed"
	else
		fail "unknown file type: $FTYPE"
	fi
	#search for bin
	DLPATH=$(find . -type f | xargs du | sort -n | tail -n 1 | cut -f 2)
	if [ ! -f "$DLPATH" ]; then
		fail "could not find downloaded binary"
	fi
	#move into PATH or cwd
	chmod +x $DLPATH || fail "chmod +x failed"
	mv $DLPATH $OUT_DIR/$PROG || fail "mv failed" #FINAL STEP!
	echo "{{ if .MoveToPath }}Installed at{{ else }}Downloaded to{{ end }} $OUT_DIR/$PROG"
	#done
	cleanup
}
install
