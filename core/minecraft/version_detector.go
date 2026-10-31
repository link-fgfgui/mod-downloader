package minecraft

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf16"
)

const (
	officialVersionJSON  = "version.json"
	clientMinecraftClass = "net/minecraft/client/Minecraft.class"
	serverMinecraftClass = "net/minecraft/server/MinecraftServer.class"
)

var serverVersionPattern = regexp.MustCompile(`^(?:Alpha |Beta )?(?:v)?[0-9]+(?:\.[0-9]+)*(?:_[0-9]+)?[a-z]?$`)

func DetectMinecraftVersionFromJar(jarPath string) (string, bool) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return "", false
	}
	defer r.Close()

	return DetectMinecraftVersionFromZip(&r.Reader)
}

func DetectMinecraftVersionFromZip(r *zip.Reader) (string, bool) {
	if version := detectMinecraftVersionJSON(r); version != "" {
		return version, true
	}
	if version := detectMinecraftClientVersion(r); version != "" {
		return version, true
	}
	if version := detectMinecraftServerVersion(r); version != "" {
		return version, true
	}
	return "", false
}

func detectMinecraftVersionJSON(r *zip.Reader) string {
	data, err := readZipFile(r, officialVersionJSON)
	if err != nil {
		return ""
	}

	var version struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &version); err != nil {
		return ""
	}
	return strings.TrimSpace(version.ID)
}

func detectMinecraftClientVersion(r *zip.Reader) string {
	data, err := readZipFile(r, clientMinecraftClass)
	if err != nil {
		return ""
	}
	strings, err := ParseClassStrings(data)
	if err != nil {
		return ""
	}
	return ExtractClientMinecraftVersion(strings)
}

func detectMinecraftServerVersion(r *zip.Reader) string {
	data, err := readZipFile(r, serverMinecraftClass)
	if err != nil {
		return ""
	}
	strings, err := ParseClassStrings(data)
	if err != nil {
		return ""
	}
	return ExtractServerMinecraftVersion(strings)
}

func ParseClassStrings(data []byte) ([]string, error) {
	if len(data) < 10 {
		return nil, io.ErrUnexpectedEOF
	}
	if binary.BigEndian.Uint32(data[:4]) != 0xCAFEBABE {
		return nil, errors.New("invalid class magic")
	}

	offset := 8
	count := int(binary.BigEndian.Uint16(data[offset:])) - 1
	offset += 2
	if count < 0 {
		return nil, errors.New("invalid constant pool count")
	}

	utf8Entries := make(map[int]string, count)
	stringRefs := make([]int, 0)

	readU2 := func() (uint16, error) {
		if offset+2 > len(data) {
			return 0, io.ErrUnexpectedEOF
		}
		value := binary.BigEndian.Uint16(data[offset:])
		offset += 2
		return value, nil
	}
	skip := func(n int) error {
		if offset+n > len(data) {
			return io.ErrUnexpectedEOF
		}
		offset += n
		return nil
	}

	for i := 0; i < count; i++ {
		if offset >= len(data) {
			return nil, io.ErrUnexpectedEOF
		}
		tag := data[offset]
		offset++

		switch tag {
		case 1:
			length, err := readU2()
			if err != nil {
				return nil, err
			}
			if offset+int(length) > len(data) {
				return nil, io.ErrUnexpectedEOF
			}
			s, err := decodeModifiedUTF8(data[offset : offset+int(length)])
			if err != nil {
				return nil, err
			}
			utf8Entries[i] = s
			offset += int(length)
		case 8:
			ref, err := readU2()
			if err != nil {
				return nil, err
			}
			stringRefs = append(stringRefs, int(ref)-1)
		case 5, 6:
			if err := skip(8); err != nil {
				return nil, err
			}
			i++
		case 3, 4, 9, 10, 11, 12, 17, 18:
			if err := skip(4); err != nil {
				return nil, err
			}
		case 7, 16, 19, 20:
			if err := skip(2); err != nil {
				return nil, err
			}
		case 15:
			if err := skip(3); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown constant pool tag: %d", tag)
		}
	}

	out := make([]string, 0, len(stringRefs))
	for _, ref := range stringRefs {
		if s, ok := utf8Entries[ref]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// ExtractClientMinecraftVersion implements the early-client class literal
// strategy documented for roughly Alpha 1.0.6 through Beta 1.7.3.
func ExtractClientMinecraftVersion(strs []string) string {
	const prefix = "Minecraft Minecraft "
	for _, s := range strs {
		raw, ok := strings.CutPrefix(s, prefix)
		if !ok {
			continue
		}
		switch {
		case raw == "RC1" || raw == "RC2":
			return ""
		case strings.HasPrefix(raw, "Beta "):
			return "b" + strings.TrimPrefix(raw, "Beta ")
		case strings.HasPrefix(raw, "Alpha v"):
			return "a" + strings.TrimPrefix(raw, "Alpha v")
		default:
			return raw
		}
	}
	return ""
}

func ExtractServerMinecraftVersion(strs []string) string {
	const maxServerVersionLookback = 8

	idx := -1
	for i, s := range strs {
		if strings.HasPrefix(s, "Can't keep up!") {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}

	start := idx - maxServerVersionLookback
	if start < 0 {
		start = 0
	}
	for i := idx - 1; i >= start; i-- {
		if serverVersionPattern.MatchString(strings.TrimSpace(strs[i])) {
			return strs[i]
		}
	}
	return ""
}

func decodeModifiedUTF8(data []byte) (string, error) {
	codeUnits := make([]uint16, 0, len(data))
	for i := 0; i < len(data); {
		b := data[i]
		switch {
		case b == 0:
			return "", errors.New("invalid modified utf-8: bare null byte")
		case b < 0x80:
			codeUnits = append(codeUnits, uint16(b))
			i++
		case b&0xE0 == 0xC0:
			if i+1 >= len(data) || !isContinuationByte(data[i+1]) {
				return "", errors.New("invalid modified utf-8: malformed two-byte sequence")
			}
			r := rune(b&0x1F)<<6 | rune(data[i+1]&0x3F)
			codeUnits = append(codeUnits, uint16(r))
			i += 2
		case b&0xF0 == 0xE0:
			if i+2 >= len(data) || !isContinuationByte(data[i+1]) || !isContinuationByte(data[i+2]) {
				return "", errors.New("invalid modified utf-8: malformed three-byte sequence")
			}
			r := rune(b&0x0F)<<12 | rune(data[i+1]&0x3F)<<6 | rune(data[i+2]&0x3F)
			codeUnits = append(codeUnits, uint16(r))
			i += 3
		default:
			return "", errors.New("invalid modified utf-8: unsupported leading byte")
		}
	}
	return string(utf16.Decode(codeUnits)), nil
}

func isContinuationByte(b byte) bool {
	return b&0xC0 == 0x80
}
