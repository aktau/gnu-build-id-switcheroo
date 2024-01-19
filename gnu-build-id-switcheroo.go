package main

import (
	"bytes"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := rmain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

func usage(format string, args ...any) error {
	return fmt.Errorf("error: "+format+"usage:\n\t"+os.Args[0]+" [new-build-id] < <elf-binary>", args...)
}

func rmain() error {
	// Read all of standard input (can't stream, debug/elf requires an
	// io.ReaderAt).
	//
	// TODO(aktau): mmap(2) it and write in-place.
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	r := bytes.NewReader(b)
	id, err := buildID(r)
	if err != nil {
		return err
	}
	// return d[12+an : 12+an+descsz], nil
	pos := bytes.Index(b, id)
	encoded := hex.EncodeToString(id)
	fmt.Fprintf(os.Stderr, "FOUND: %s (%d bytes) at offset %d\n", encoded, len(id), pos)

	if len(os.Args) > 2 {
		return usage("too many argument, expected 1 (read-only) or 2 (replace), got %d", len(os.Args))
	}

	// Replace the note.
	//
	// The debug/elf API does not provide writing, so we perform the dirtiest of
	// hacks: find the byte pattern corresponding to the found tag, and change it
	// in place.
	if len(os.Args) == 2 {
		replHex := os.Args[1]
		fmt.Fprintf(os.Stderr, "Replacing with : %s\n", replHex)
		newID, err := hex.DecodeString(replHex)
		if err != nil {
			return usage("%q does not appear to be a valid hex string: %v", replHex, err)
		}
		if len(newID) != len(id) {
			return usage("new build ID should have the same size as the old one, old: %d, new: %d (%q)", len(id), len(newID), replHex)
		}
		if matches := bytes.Count(b, id); matches != 1 {
			return usage("found %d matches of %q in the binary, not replacing (BUG, make this more precise)", matches)
		}
		copy(b[pos:pos+len(id)], newID)
		_, err = io.Copy(os.Stdout, bytes.NewReader(b))
		return err
	}

	return nil
}

func buildID(r io.ReaderAt) ([]byte, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}

	for i, s := range f.Sections {
		if s.Type != elf.SHT_NOTE {
			continue
		}

		d, err := s.Data()
		if err != nil {
			return nil, fmt.Errorf("could not read data in section %v: %v", elf.SHT_NOTE, err)
		}
		for len(d) > 0 {
			// ELF standards differ as to the sizes in note sections. Both the GNU
			// linker and gold always generate 32-bit sizes, so that is what we assume
			// here.
			if len(d) < 12 {
				return nil, fmt.Errorf("note section %d too short (%d<12)", i, len(d))
			}
			namesz := f.ByteOrder.Uint32(d[0:4])
			descsz := f.ByteOrder.Uint32(d[4:8])
			typ := f.ByteOrder.Uint32(d[8:12])
			an := (namesz + 3) &^ 3 // (namesz + (alignment-1)) &^ (alignment-1)
			ad := (descsz + 3) &^ 3

			if int(12+an+ad) > len(d) {
				return nil, fmt.Errorf("note section %d too short for header (%d < 12 + align(%d,4) + align(%d,4))", i, len(d), namesz, descsz)
			}

			// 3 == NT_GNU_BUILD_ID
			if typ == 3 && namesz == 4 && bytes.Equal(d[12:16], []byte("GNU\000")) {
				return d[12+an : 12+an+descsz], nil
			}
			d = d[12+an+ad:]
		}
	}
	return nil, fmt.Errorf("NT_GNU_BUILD_ID not found")
}
