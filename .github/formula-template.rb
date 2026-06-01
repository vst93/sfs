class Sfs < Formula
  desc "SmallFileSync - A WebDAV-based terminal file sync tool"
  homepage "https://github.com/vst93/sfs"
  version "{{VERSION}}"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/vst93/sfs/releases/download/v{{VERSION}}/sfs-darwin-arm64.zip"
      sha256 "{{SHA256_DARWIN_ARM64}}"
    else
      url "https://github.com/vst93/sfs/releases/download/v{{VERSION}}/sfs-darwin-amd64.zip"
      sha256 "{{SHA256_DARWIN_AMD64}}"
    end
  end

  if OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/vst93/sfs/releases/download/v{{VERSION}}/sfs-linux-arm64.zip"
      sha256 "{{SHA256_LINUX_ARM64}}"
    else
      url "https://github.com/vst93/sfs/releases/download/v{{VERSION}}/sfs-linux-amd64.zip"
      sha256 "{{SHA256_LINUX_AMD64}}"
    end
  end

  def install
    bin.install "sfs"
  end

  test do
    system "#{bin}/sfs", "--version"
  end
end
