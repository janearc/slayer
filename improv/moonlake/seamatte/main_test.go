package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandProcess(t *testing.T) {
	if os.Getenv("SEAMATTE_TEST_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"seamatte"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func command(t *testing.T, args []string, vars map[string]string) ([]byte, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCommandProcess$", "--"}, args...)...)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "MOONREF=") && !strings.HasPrefix(entry, "WATERMASK=") && !strings.HasPrefix(entry, "FRONT=") && !strings.HasPrefix(entry, "SEAMATTE_TEST_PROCESS=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "SEAMATTE_TEST_PROCESS=1")
	for key, value := range vars {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return out, exit.ExitCode()
	}
	t.Fatal(err)
	return nil, -1
}

func fixture(t *testing.T, dir, name string, width, height int, scale float64, mode int) string {
	t.Helper()
	var data bytes.Buffer
	fmt.Fprintf(&data, "PF\n%d %d\n%g\n", width, height, scale)
	pixels := make([]float32, width*height*3)
	for i := range pixels {
		switch mode {
		case 0:
			pixels[i] = float32(i%19+1) / 7
		case 1:
			pixels[i] = float32(i%11+2) / 13
		case 2:
			pixels[i] = float32(i%7) / 6
		case 3:
			pixels[i] = float32(i%5) / 17
		case 4:
			pixels[i] = float32(i%13+1) * 3
		}
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	if err := binary.Write(&data, order, pixels); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCommandBytes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scale    float64
		optional bool
		want     string
	}{
		{"little-shore", -1.25, false, "2e9c62612cbb0ffbcd18ccf672738e4948b62250fd921dd9117bcf18901961ee"},
		{"big-shore", 2.5, false, "56160f32bdb0719435a0ec28cb2d585de75d34883ca161bbe6f96fb951ca9593"},
		{"little-layers", -1.25, true, "206ca925c276cc61199d68a8a13e1ebfcc0cf24a09f14b5d4aab631bceddb2c1"},
		{"big-layers", 2.5, true, "c98b7ccaaa02a221c98e0f109bff3461e4eda297ceef0c335d2610fc48ead914"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw := fixture(t, dir, "raw.pfm", 2, 12, tc.scale, 0)
			sea := fixture(t, dir, "sea.pfm", 2, 12, -1, 1)
			out := filepath.Join(dir, "out.pfm")
			vars := map[string]string{}
			if tc.optional {
				vars["MOONREF"] = fixture(t, dir, "ref.pfm", 1, 3, 1, 4)
				vars["WATERMASK"] = fixture(t, dir, "mask.pfm", 2, 12, 1, 2)
				vars["FRONT"] = fixture(t, dir, "front.pfm", 2, 12, -1, 3)
			}
			text, status := command(t, []string{raw, sea, out, "6", "-6", "0", "0.27"}, vars)
			if status != 0 || len(text) != 0 {
				t.Fatalf("status %d, output %q", status, text)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(data))
			if got != tc.want {
				t.Fatalf("output hash %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCommandErrors(t *testing.T) {
	dir := t.TempDir()
	raw := fixture(t, dir, "raw.pfm", 2, 12, -1, 0)
	sea := fixture(t, dir, "sea.pfm", 2, 12, -1, 1)
	small := fixture(t, dir, "small.pfm", 1, 1, -1, 1)
	missing := filepath.Join(dir, "missing.pfm")
	bad := filepath.Join(dir, "bad.pfm")
	if err := os.WriteFile(bad, []byte("Pf\n1 1\n-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	short := filepath.Join(dir, "short.pfm")
	if err := os.WriteFile(short, []byte("PF\n1 1\n-1\nx"), 0600); err != nil {
		t.Fatal(err)
	}
	base := []string{raw, sea, filepath.Join(dir, "out.pfm"), "6", "-6", "0", "0.27"}
	for _, tc := range []struct {
		name   string
		index  int
		value  string
		vars   map[string]string
		status int
		want   string
	}{
		{"usage", -1, "", nil, 2, "seamatte raw.pfm sea.pfm out.pfm top bottom shore moonradius\n"},
		{"number", 3, "bad", nil, 1, "seamatte: strconv.ParseFloat: parsing \"bad\": invalid syntax\n"},
		{"raw", 0, missing, nil, 1, "seamatte: open " + missing + ": no such file or directory\n"},
		{"sea", 1, missing, nil, 1, "seamatte: open " + missing + ": no such file or directory\n"},
		{"dimensions", 1, small, nil, 1, "seamatte: the raw is 2x12 and the sea 1x1\n"},
		{"kind", 0, bad, nil, 1, "seamatte: " + bad + " is not a colour pfm\n"},
		{"truncated", 0, short, nil, 1, "seamatte: unexpected EOF\n"},
		{"reference", -2, "", map[string]string{"MOONREF": missing}, 1, "seamatte: open " + missing + ": no such file or directory\n"},
		{"mask", -2, "", map[string]string{"WATERMASK": missing}, 1, "seamatte: open " + missing + ": no such file or directory\n"},
		{"front", -2, "", map[string]string{"FRONT": missing}, 1, "seamatte: open " + missing + ": no such file or directory\n"},
		{"output", 2, dir, nil, 1, "seamatte: open " + dir + ": is a directory\n"},
		{"configuration", 3, "NaN", nil, 1, "seamatte: nonfinite matte configuration\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string(nil), base...)
			if tc.index == -1 {
				args = nil
			} else if tc.index >= 0 {
				args[tc.index] = tc.value
			}
			got, status := command(t, args, tc.vars)
			if status != tc.status || string(got) != tc.want {
				t.Fatalf("status %d, output %q; want status %d, output %q", status, got, tc.status, tc.want)
			}
		})
	}
}
