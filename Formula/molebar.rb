class Molebar < Formula
  desc "macOS menu bar widget for Mole system metrics"
  homepage "https://github.com/ayushkumar912/MoleBar"
  url "https://github.com/ayushkumar912/MoleBar/releases/download/v0.1.3/molebar-0.1.3.tar.gz"
  sha256 "55b00edfeca93f24eb57a6164d233a315480db1e48ba6364865df723f3ce9385"
  license "Apache-2.0"
  head "https://github.com/ayushkumar912/MoleBar.git", branch: "main"

  livecheck do
    url :stable
    regex(/^v?(\d+(?:\.\d+)+)$/i)
  end

  depends_on "go" => :build
  depends_on macos: :big_sur
  depends_on "mole"

  def install
    system "make", "app", "VERSION=#{version}"
    prefix.install "build/MoleBar.app"
    bin.write_exec_script prefix/"MoleBar.app/Contents/MacOS/molebar"
  end

  def caveats
    <<~EOS
      MoleBar.app is installed to:
        #{opt_prefix}/MoleBar.app

      Launch the menu bar widget with:
        open #{opt_prefix}/MoleBar.app
      or run `molebar` from a terminal.
    EOS
  end

  test do
    output = shell_output("#{bin}/molebar -interval=0 2>&1", 2)
    assert_match "interval must be greater than 0", output
    assert_path_exists prefix/"MoleBar.app/Contents/MacOS/molebar"
    assert_match version.to_s, File.read(prefix/"MoleBar.app/Contents/Info.plist")
  end
end
