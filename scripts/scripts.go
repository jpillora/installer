package scripts

import _ "embed"

//go:embed install.txt.tmpl
var Text []byte

//go:embed install.sh.tmpl
var Shell []byte

//go:embed install.rb.tmpl
var Homebrew []byte

//go:embed install.ps1.tmpl
var Powershell []byte
