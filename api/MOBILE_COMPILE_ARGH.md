# Argh

So far my glorious plan in README.md to compile this C API for mobile apps has been unsuccessful.

Here's the trail of tears as a note to myself or whoever:

- Go has, famously, cross-compiling built-in to the standard compiler
  - But it doesn't work if you use cgo
- But, there's [xgo](https://github.com/karalabe/xgo) which looks like EXACTLY what you'd need
  - But it [doesn't work](https://github.com/karalabe/xgo/issues/138) with the new Golang module system, which Replicant uses
- But [there's a fork](https://github.com/karalabe/xgo/issues/138#issuecomment-454946751) that claims to have module support
  - But you need to build your own Docker image, and the build relies on downloading Apple sdks from a website called "sdks.website" which is down.
- OK this is starting to feel pretty sketch.... rewind
- We can work around the lack of modules support
  - Use `go mod vendor` to populate the vendor directory of replicant
  - Symlink replicant dir to `$GOPATH/src/github.com/aboodman/replicant`
- Now `xgo` works!
  - ~/src/github.com/karalabe/xgo/xgo --targets=ios/*,android/* ./api
  - It builds a `.framework` and `.aar` file
  - And for iOS in particular, the framework includes builds for arm7 and arm64
- But, for iOS, it doesn't build an x86 version for the simulator
  - The simulator build is crucial because developers typically develop first on it, then test on phones later
- The instructions in xgo [mention this problem](https://github.com/karalabe/xgo#mobile-libraries):
  - I was able to find iOS Simulator SDK v9.3 on dev.apple.com
    - Amusingly part of the XCode 7.3.1
    - But after a bunch of work hacking xgo, I still did not succeed in creating a working build
- Le sigh. Rewinding again.
- It is actually possible to just build the simulator version without xgo on a mac, because cross-compiling isn't required:
  - `go build -tags=ios -buildmode=c-archive -o api.a api.go`
  - This produces an .a file you can just drop in xcode and it works!
  - However:
    - Need to find a way to get it into the framework, so that there's a single distributable
    - And means build has to happen on a mac. I guess that's probably doable.

# Potential paths forward

- Build the simulator version outside xgo and patch it into a framework manually
- Look at what gomobile does
  - gomobile is a fully-integrated solution to make go work on ios/android, but it generates bindings, which I don't want
  - still it must be generating simulator builds somehow
- Or just give up on this craziness and use gomobile
