package git

import (
	"bytes"
	"errors"
	"os/exec"
)

// FilesContainingWords returns the tracked files under root, relative to root
// and including submodule content, that contain any of words as a whole word, skipping binary files and the
// root-relative exclude paths. A search without matches returns nil and no
// error; an error means git is unavailable or root is not in a work tree.
func FilesContainingWords(root string, words, exclude []string) ([]string, error) {
	if len(words) == 0 {
		return nil, nil
	}
	args := []string{"-C", root, "grep", "--recurse-submodules", "-z", "-l", "-I", "-w", "-F"}
	for _, w := range words {
		args = append(args, "-e", w)
	}
	args = append(args, "--", ".")
	for _, p := range exclude {
		args = append(args, ":(exclude)"+p)
	}

	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// git grep exits 1 when nothing matches.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range bytes.Split(out, []byte{0}) {
		if len(entry) > 0 {
			files = append(files, string(entry))
		}
	}
	return files, nil
}
