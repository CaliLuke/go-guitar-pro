# Compatibility Corpus

These directories contain Guitar Pro files for parser compatibility tests. Each directory matches one format version of Guitar Pro.

The corpus contains valid files and known unsupported fixtures. `goguitarpro_test.go` lists the unsupported fixtures. If the parser does not support a fixture, keep the fixture in the corpus.

Many fixtures originate from the alphaTab test suite. See `THIRD_PARTY_NOTICES.md` and `LICENSE` for their terms.
