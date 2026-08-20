# Possible future work

A working product can always be pushed further. This file keeps potentially useful ideas visible without turning them into commitments: an improvement should be implemented when it solves a concrete need, not simply because it is technically possible.

## To consider

- **User-facing timing controls** — dwell time is currently a runtime override. It could become a normal setting later, together with sprite animation speed.

- **Built-in sprites** — add complete sprite sets to the Default Gopher Pack.

- **More tests** — the current test suite focuses on the parts where state, fallback and filesystem edge cases matter most. It could later be extended to other parts of the application.

- **GitHub Releases** — define a proper release process: version tags, `.app` archives, Apple Silicon / Intel builds if needed, and eventually automated release builds.

- **Signing and notarization** — investigate proper macOS signing/notarization if direct distribution through GitHub Releases becomes the normal installation method.

- **Mac App Store** — evaluate whether App Store distribution would actually be useful enough to justify the additional constraints.

- **More README visuals** — add a few screenshots or animations once the final UI and built-in sprites are settled.

- **Continuous integration** — add GitHub Actions to automatically run tests and builds on pushes and pull requests instead of relying only on local validation.

- **Local systray fork maintenance** — the vendored `systray` fork solves real macOS-specific needs, but it also becomes part of GopherTchi's maintenance surface. Keep track of upstream fixes and minimize divergence where practical.

- **Production-oriented robustness** — as the project grows, consider whether additional logging, diagnostics, failure recovery, compatibility testing or performance checks become useful. These should be added only when the application's complexity actually justifies them.
