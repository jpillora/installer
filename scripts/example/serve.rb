require "formula"

class Serve < Formula
  homepage "https://github.com/jpillora/serve"
  version "1.7.2"

  if !OS.linux? && !Hardware.is_64_bit?
    url "https://github.com/jpillora/serve/releases/download/1.7.2/serve_darwin_386.gz"
  elsif !OS.linux? && Hardware.is_64_bit?
    url "https://github.com/jpillora/serve/releases/download/1.7.2/serve_darwin_amd64.gz"
    sha1 "b19b8a57925f5f51ea671f4919856fa470ef9832"
  elsif OS.linux? && !Hardware.is_64_bit?
    url "https://github.com/jpillora/serve/releases/download/1.7.2/serve_linux_386.gz"
  elsif OS.linux? && Hardware.is_64_bit?
    url "https://github.com/jpillora/serve/releases/download/1.7.2/serve_linux_amd64.gz"
  else
    onoe "Not supported"
  end

  depends_on :arch => :intel

  def install
    bin.install Dir["*"][0]
    system "mv","#{bin}/serve_darwin_amd64","#{bin}/serve"
  end

  def caveats
    "serve was installed using github.com/jpillora/installer"
  end
end
