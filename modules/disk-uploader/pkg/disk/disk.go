package disk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

var progressRegex = regexp.MustCompile(`\((\d+(?:\.\d+)?)\/100%\)`)

// progressWriter wraps an io.Writer and filters qemu-img progress output
// to only emit updates when the whole-number percentage changes.
type progressWriter struct {
	out            io.Writer
	lastPercentage int
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	matches := progressRegex.FindSubmatch(p)
	if matches == nil {
		return pw.out.Write(p)
	}

	val, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return pw.out.Write(p)
	}

	pct := int(val)
	if pct == pw.lastPercentage {
		return len(p), nil
	}
	pw.lastPercentage = pct
	return fmt.Fprintf(pw.out, "    (%d/100%%)\r", pct)
}

func DownloadDiskImageFromURL(rawDiskUrl, headerKey, headerValue, certificatePath, diskPath string) error {
	cmd := exec.Command(
		"nbdkit",
		"-r",
		"curl",
		rawDiskUrl,
		fmt.Sprintf("header=%s: %s", headerKey, headerValue),
		fmt.Sprintf("cainfo=%s", certificatePath),
		"--run",
		fmt.Sprintf("qemu-img convert \"$uri\" -p -O qcow2 %s", diskPath),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = &progressWriter{out: os.Stderr, lastPercentage: -1}

	if err := cmd.Run(); err != nil {
		return err
	}

	if fileInfo, err := os.Stat(diskPath); err != nil || fileInfo.Size() == 0 {
		return fmt.Errorf("disk image file does not exist or is empty")
	}
	return nil
}
