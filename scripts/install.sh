#!/bin/bash
TMP_DIR="/tmp/tmpinstalldir"
function cleanup {
	echo rm -rf $TMP_DIR > /dev/null
}
function fail {
	cleanup
	msg=$1
	echo "============"
	echo "Error: $msg" 1>&2
	exit 1
}
function os_release() {
if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
	source /etc/os-release
        DISTRO=${ID}
        DISTRO_VERSION=${VERSION_ID}
fi
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
	OS="linux"
	# Can keep for extra tasks on linux
	if { [ "${DISTRO}" == "ubuntu" ] || [ "${DISTRO}" == "debian" ] || [ "${DISTRO}" == "raspbian" ] || [ "${DISTRO}" == "pop" ] || [ "${DISTRO}" == "kali" ] || [ "${DISTRO}" == "linuxmint" ] || [ "${DISTRO}" == "fedora" ] || [ "${DISTRO}" == "centos" ] || [ "${DISTRO}" == "rhel" ] || [ "${DISTRO}" == "arch" ] || [ "${DISTRO}" == "archarm" ] || [ "${DISTRO}" == "manjaro" ] || [ "${DISTRO}" == "alpine" ] || [ "${DISTRO}" == "freebsd" ] || [ "${DISTRO}" == "neon" ] || [ "${DISTRO}" == "almalinux" ] || [ "${DISTRO}" == "rocky" ]; }; then
	echo "$DISTRO"
		elif { [ "${DISTRO}" == "fedora" ] || [ "${DISTRO}" == "centos" ] || [ "${DISTRO}" == "rhel" ] || [ "${DISTRO}" == "almalinux" ] || [ "${DISTRO}" == "rocky" ]; }; then
	echo "$DISTRO"
		elif { [ "${DISTRO}" == "arch" ] || [ "${DISTRO}" == "archarm" ] || [ "${DISTRO}" == "manjaro" ]; }; then
	echo "$DISTRO"
		elif { [ "${DISTRO}" == "fedora" ] || [ "${DISTRO}" == "centos" ] || [ "${DISTRO}" == "rhel" ] || [ "${DISTRO}" == "almalinux" ] || [ "${DISTRO}" == "rocky" ]; }; then
	echo "$DISTRO"
		elif { [ "${DISTRO}" == "arch" ] || [ "${DISTRO}" == "archarm" ] || [ "${DISTRO}" == "manjaro" ]; }; then
	echo "$DISTRO"
		elif [ "${DISTRO}" == "alpine" ]; then
	echo "$DISTRO"
	fi
fi
if [[ "$OSTYPE" == "darwin"* ]]; then
	# Seperating as most likely darwin will be expanded.
	OS="darwin"
fi
if [[ "$OSTYPE" == "cygwin" ]]; then
	# POSIX compatibility layer and Linux environment emulation for Windows
	echo "Your Operating System not supported yet..."
		elif [[ "$OSTYPE" == "msys" ]]; then
	# Lightweight shell and GNU utilities compiled for Windows (part of MinGW)
	echo "Your Operating System not supported yet..."
		elif [[ "$OSTYPE" == "win32" ]]; then
	# Not sure
	echo "Your Operating System not supported yet..."
fi
if { [[ "$OSTYPE" == "freebsd"* ]] || [[ "$DISTRO" == "freebsd" ]]; }; then
        # Quite confident freebsd
	OS="freebsd"
fi
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
	[ ! "$BASH_VERSION" ] && fail "Please use bash instead"
	[ ! -d $OUT_DIR ] && fail "output directory missing: $OUT_DIR"
	#dependency check, assume we are a standard POISX machine
	which find > /dev/null || fail "find not installed"
	which xargs > /dev/null || fail "xargs not installed"
	which sort > /dev/null || fail "sort not installed"
	which tail > /dev/null || fail "tail not installed"
	which cut > /dev/null || fail "cut not installed"
	which du > /dev/null || fail "du not installed"
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
	os_release
	#find ARCH
	if uname -m | grep 64 > /dev/null; then
		ARCH="amd64"
	elif uname -m | grep arm > /dev/null; then
		ARCH="arm" #TODO armv6/v7
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
	#got URL! download it...
	echo -n "{{ if .MoveToPath }}Installing{{ else }}Downloading{{ end }} $USER/$PROG $RELEASE"
	{{ if .Google }}
	#matched using google, give time to cancel
	echo -n " in 5 seconds"
	for i in 1 2 3 4 5; do
		sleep 1
		echo -n "."
	done
	{{ else }}
	echo "....."
	{{ end }}
	#enter tempdir
	mkdir -p $TMP_DIR
	cd $TMP_DIR
	if [[ $FTYPE = ".gz" ]]; then
		which gzip > /dev/null || fail "gzip is not installed"
		#gzipped binary
		NAME="${PROG}_${OS}_${ARCH}.gz"
		GZURL="$GH/releases/download/$RELEASE/$NAME"
		#gz download!
		bash -c "$GET $URL" | gzip -d - > $PROG || fail "download failed"
	elif [[ $FTYPE = ".tar.gz" ]] || [[ $FTYPE = ".tgz" ]]; then
		#check if archiver progs installed
		which tar > /dev/null || fail "tar is not installed"
		which gzip > /dev/null || fail "gzip is not installed"
		bash -c "$GET $URL" | tar zxf - || fail "download failed"
	elif [[ $FTYPE = ".zip" ]]; then
		which unzip > /dev/null || fail "unzip is not installed"
		bash -c "$GET $URL" > tmp.zip || fail "download failed"
		unzip -o -qq tmp.zip || fail "unzip failed"
		rm tmp.zip || fail "cleanup failed"
	elif [[ $FTYPE = "" ]]; then
		bash -c "$GET $URL" > "{{ .Program }}_${OS}_${ARCH}" || fail "download failed"
	else
		fail "unknown file type: $FTYPE"
	fi
	#search subtree largest file (bin)
	TMP_BIN=$(find . -type f | xargs du | sort -n | tail -n 1 | cut -f 2)
	if [ ! -f "$TMP_BIN" ]; then
		fail "could not find find binary (largest file)"
	fi
	#ensure its larger than 1MB
	if [[ $(du -m $TMP_BIN | cut -f1) -lt 1 ]]; then
		fail "no binary found ($TMP_BIN is not larger than 1MB)"
	fi
	#move into PATH or cwd
	chmod +x $TMP_BIN || fail "chmod +x failed"
	{{ if .SudoMove }}echo "using sudo to move binary..."{{ end }}
	{{ if .SudoMove }}sudo {{ end }}mv $TMP_BIN $OUT_DIR/$PROG || fail "mv failed" #FINAL STEP!
	echo "{{ if .MoveToPath }}Installed at{{ else }}Downloaded to{{ end }} $OUT_DIR/$PROG"
	#done
	cleanup
}
install
