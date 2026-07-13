package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/3cpo-dev/flawless/internal/gitx"
	"github.com/3cpo-dev/flawless/internal/run"
)

func cmdLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	follow := fs.Bool("f", false, "follow the log while the run is active")
	full := fs.Bool("full", false, "print the whole log instead of the tail")
	lines := fs.Int("n", 40, "tail line count")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoDir, err := gitx.RepoRoot(".")
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	r, err := run.Latest(repoDir)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("no runs yet — start one with: flawless")
	}

	f, err := os.Open(r.LogPath())
	if err != nil {
		return fmt.Errorf("no log for run %s", r.ID)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if *full || *follow {
		os.Stdout.Write(data)
	} else {
		content := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		if len(content) > *lines {
			content = content[len(content)-*lines:]
		}
		fmt.Println(strings.Join(content, "\n"))
	}
	if !*follow {
		return nil
	}
	// Follow by polling for appended bytes until the run's process exits.
	// (No daemon to subscribe to — the file is the interface.)
	offset := int64(len(data))
	for {
		if !r.Alive() {
			// Read whatever landed after the last poll, then stop.
			printFrom(f, &offset)
			fresh, _ := run.Latest(repoDir)
			if fresh != nil {
				fmt.Printf("-- run %s: %s --\n", fresh.ID, effectiveStatus(fresh))
			}
			return nil
		}
		time.Sleep(500 * time.Millisecond)
		printFrom(f, &offset)
		if fresh, _ := run.Latest(repoDir); fresh != nil {
			*r = *fresh
		}
	}
}

func printFrom(f *os.File, offset *int64) {
	if _, err := f.Seek(*offset, io.SeekStart); err != nil {
		return
	}
	n, _ := io.Copy(os.Stdout, f)
	*offset += n
}
