package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

func parseAnnouncementFile(path string) (altName string, message string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", "", err
	}
	s := string(b)

	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "X-Wii-AltName:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				altNameB64 := strings.TrimSpace(parts[1])
				altBytes, _ := base64.StdEncoding.DecodeString(altNameB64)
				altName = decodeUTF16BE(altBytes)
				break
			}
		}
	}

	idx := strings.Index(s, "Content-Type: text/plain; charset=utf-16be")
	rest := s[idx:]
	r := bufio.NewReader(strings.NewReader(rest))
	foundBlank := false
	var payloadLines []string
	for {
		l, errr := r.ReadString('\n')
		if errr != nil && errr != io.EOF {
			break
		}
		trim := strings.TrimRight(l, "\r\n")
		if !foundBlank {
			if trim == "" {
				foundBlank = true
			}
		} else {
			if strings.HasPrefix(trim, "----") || strings.HasPrefix(trim, "--") {
				break
			}
			if strings.HasPrefix(trim, "Content-") || strings.HasPrefix(trim, "Content-Type:") || strings.HasPrefix(trim, "Content-Transfer-Encoding:") {
			} else if trim != "" {
				payloadLines = append(payloadLines, trim)
			}
		}
		if errr == io.EOF {
			break
		}
	}

	if len(payloadLines) == 0 {
		return altName, "", nil
	}

	payload := strings.Join(payloadLines, "")
	decoded, derr := base64.StdEncoding.DecodeString(payload)
	if derr != nil {
		return altName, "", derr
	}
	message = decodeUTF16BE(decoded)
	return altName, message, nil
}

func decodeUTF16BE(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	u16 := make([]uint16, len(b)/2)
	for i := 0; i < len(u16); i++ {
		u16[i] = binary.BigEndian.Uint16(b[i*2:])
	}
	runes := utf16.Decode(u16)
	return string(runes)
}
