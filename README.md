gnu-build-id-switcheroo
=======================

A small program that replaces the GNU build ID in elf binaries with another of
equal length.

Usage
-----

Replace whatever the current build ID is (if it is set and of equal length),
with `3b3788c8b2f7d44a91cf179bb8f02bac9dbef093`.

    gnu-build-id-switcheroo 3b3788c8b2f7d44a91cf179bb8f02bac9dbef093 < mybinary > mybinary.mod
