require "formula"

class Serve < Formula
  homepage "https://github.com/jpillora/serve"
  version "1.7.2"
  if !OS.linux? && Hardware.is_64_bit?
    url "https://github.com/jpillora/serve/releases/download/1.7.2/serve_darwin_amd64.gz"
    # md5 "d53e75f2493a7557f086046d66502ed1" MD5 NO WORK, MUST DOWNLOAD WHOLE FILE TO CALC THIS
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
