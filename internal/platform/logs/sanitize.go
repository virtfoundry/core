package logs

import (
	"io"
	"regexp"
	"strings"
)

var stripPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)virt-launcher-[a-z0-9-]+`),
	regexp.MustCompile(`(?i)pods? "[^"]+"`),
	regexp.MustCompile(`(?i)namespace "[^"]+"`),
	regexp.MustCompile(`(?i)system:serviceaccount:[^\s"]+`),
	regexp.MustCompile(`(?i)kubevirt\.io[^\s]*`),
	regexp.MustCompile(`(?i)network-attachment-definitions[^\s]*`),
	regexp.MustCompile(`(?i)cannot get resource "[^"]+"`),
}

// SanitizeReader strips infrastructure details from log streams.
func SanitizeReader(src io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		buf := make([]byte, 4096)
		var carry strings.Builder
		for {
			n, err := src.Read(buf)
			if n > 0 {
				carry.Write(buf[:n])
				text := carry.String()
				lastNL := strings.LastIndex(text, "\n")
				if lastNL >= 0 {
					chunk := text[:lastNL+1]
					carry.Reset()
					carry.WriteString(text[lastNL+1:])
					for _, line := range strings.Split(chunk, "\n") {
						if line == "" {
							continue
						}
						pw.Write([]byte(sanitizeLine(line) + "\n"))
					}
				}
			}
			if err != nil {
				if carry.Len() > 0 {
					pw.Write([]byte(sanitizeLine(carry.String())))
				}
				break
			}
		}
	}()
	return pr
}

func SanitizeText(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			out.WriteString("\n")
			continue
		}
		out.WriteString(sanitizeLine(line))
		out.WriteString("\n")
	}
	return strings.TrimSuffix(out.String(), "\n")
}

func sanitizeLine(line string) string {
	for _, re := range stripPatterns {
		line = re.ReplaceAllString(line, "")
	}
	line = strings.TrimSpace(line)
	return line
}
