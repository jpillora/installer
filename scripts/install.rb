require "formula"

class In{{ .Release }} < Formula
  homepage "https://github.com/{{ .User }}/{{ .Program }}"
  version "{{ .Release }}"

  {{ range .Assets }}{{ if ne .Arch "arm" }}if {{if .IsMac }}!{{end}}OS.linux? && {{if .Is32bit }}!{{end}}Hardware.is_64_bit?
    url "{{ .URL }}"
    # sha1 "223411f5fd0f49bfef6d2e0fd34b47ad7c48fd1b"
  els{{end}}{{end}}e
    onoe "Not supported"
  end

  depends_on :arch => :intel

  def install
    bin.install '{{ .Program }}'
  end

  def caveats
    "{{ .Program }} was installed using https://github.com/jpillora/installer"
  end
end
