class Lume < Formula
  desc "Fast, safe macOS disk cleanup tool — always moves to Trash, never rm"
  homepage "https://github.com/Tyooughtul/lume"
  version "1.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Tyooughtul/lume/releases/download/v1.0.3/lume-darwin-arm64"
      sha256 "3d780131e513f5f7f4582059a3e95970fa153ca80752f745ce5d996759b4be4f"

      def install
        bin.install "lume-darwin-arm64" => "lume"
      end
    elsif Hardware::CPU.intel?
      url "https://github.com/Tyooughtul/lume/releases/download/v1.0.3/lume-darwin-amd64"
      sha256 "11a331e04fe2bd51833010d3047b38dab2acbad04d343e184e5923c6a0609fd7"

      def install
        bin.install "lume-darwin-amd64" => "lume"
      end
    end
  end

  test do
    assert_match "Lume", shell_output("#{bin}/lume -help")
  end
end
